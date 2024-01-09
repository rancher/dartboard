import { check, sleep } from 'k6'
import exec from 'k6/execution';
import http from 'k6/http';
import {Gauge} from 'k6/metrics';
import {retryOnConflict} from "./rancher_utils.js";

// Parameters
const roleCount = Number(__ENV.ROLE_COUNT)
const userCount = Number(__ENV.USER_COUNT)
const vus = Math.min(1, userCount, roleCount)
const resourcesPerRole = 5
const bindingsPerUser = 5

// Option setting
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD

// Option setting
export const options = {
    insecureSkipTLSVerify: true,

    setupTimeout: '8h',

    scenarios: {
        createRoles: {
            executor: 'shared-iterations',
            exec: 'createRoles',
            vus: vus,
            iterations: roleCount,
            maxDuration: '1h',
        },
        createUsers: {
            executor: 'shared-iterations',
            exec: 'createUsers',
            vus: vus,
            iterations: userCount,
            maxDuration: '1h',
        },
    },
    thresholds: {
        checks: ['rate>0.99']
    }
}

// Custom metrics
const resourceMetric = new Gauge('test_resources')

// Test functions, in order of execution

export function setup() {
    // log in
    const res = http.post(`${baseUrl}/v3-public/localProviders/local?action=login`, JSON.stringify({
        "description": "UI session",
        "responseType": "cookie",
        "username": username,
        "password": password
    }))

    check(res, {
        '/v3-public/localProviders/local?action=login returns status 200': (r) => r.status === 200,
    })

    const cookies = http.cookieJar().cookiesForURL(res.url)

    // delete leftovers, if any
    cleanup(cookies)

    return cookies
}

function cleanup(cookies) {
    let res = http.get(`${baseUrl}/v1/management.cattle.io.globalroles`, {cookies: cookies})
    check(res, {
        '/v1/management.cattle.io.globalroles returns status 200': (r) => r.status === 200 || r.status === 204,
    })
    JSON.parse(res.body)["data"].filter(r => r["description"].startsWith("Test ")).forEach(r => {
        res = http.del(`${baseUrl}/v3/globalRoles/${r["id"]}`, {cookies: cookies})
        check(res, {
            'DELETE /v3/globalRoles returns status 200': (r) => r.status === 200 || r.status === 204,
        })
    })
    res = http.get(`${baseUrl}/v1/management.cattle.io.users`, {cookies: cookies})
    check(res, {
        '/v1/management.cattle.io.users returns status 200': (r) => r.status === 200 || r.status === 204,
    })
    JSON.parse(res.body)["data"].filter(r => r["description"].startsWith("Test ")).forEach(r => {
        res = http.del(`${baseUrl}/v3/users/${r["id"]}`, {cookies: cookies})
        check(res, {
            'DELETE /v3/users returns status 200': (r) => r.status === 200  || r.status === 204,
        })
    })
    sleep(2)
}

const groupResources = [
    ["fleet.cattle.io", "gitrepos"],
    ["fleet.cattle.io", "bundledeployments"],
    ["apiregistration.k8s.io", "apiservices"],
    ["rbac.authorization.k8s.io", "clusterroles"],
    ["events.k8s.io","events"],
    ["apps","replicasets"],
    ["discovery.k8s.io","endpointslices"],
]

const verbs = ["create", "delete", "get", "list","patch", "update", "watch"]

export function createRoles(cookies) {
    const i = exec.scenario.iterationInTest

    const rules = Array.from({length: resourcesPerRole}, (_, j) => ({
        "apiGroups": [groupResources[(i*resourcesPerRole + j) % groupResources.length][0]],
        "nonResourceURLs": [],
        "resourceNames": [],
        "resources": [groupResources[(i*resourcesPerRole + j) % groupResources.length][1]],
        "verbs": [verbs[(i*resourcesPerRole + j) % verbs.length]]
    }))

    const res = http.post(
        `${baseUrl}/v3/globalroles`,
        JSON.stringify({
            "type": "globalRole",
            "name": `Test Global Role ${i}`,
            "description": `Test Global Role ${i}`,
            "rules": rules,
            "newUserDefault": false,
        }),
        { cookies: cookies }
    )

    check(res, {
        'v3/globalroles returns status 201': (r) => r.status === 201,
    })

    resourceMetric.add(roleCount + userCount)
}

const bindings = [
    "user", "restricted-admin", "user-base", "authn-manage", "kontainerdrivers-manage",
    "clustertemplaterevisions-create", "catalogs-use", "features-manage", "clusters-create", "catalogs-manage",
    "settings-manage", "view-rancher-metrics", "nodedrivers-manage", "clustertemplates-create",
    "podsecuritypolicytemplates-manage", "users-manage"
]

export function createUsers(cookies) {
    const i = exec.scenario.iterationInTest

    const res = http.post(`${baseUrl}/v3/users`,
        JSON.stringify({
            "type": "user",
            "name": `Test User ${i}`,
            "description": `Test User ${i}`,
            "enabled": true,
            "mustChangePassword": false,
            "password": "useruseruser",
            "username": `user-${i}`
        }),
        {cookies: cookies}
    )

    sleep(0.1)
    if (res.status != 201) {
        console.log(res)
    }
    check(res, {
        '/v3/users returns status 201': (r) => r.status === 201,
    })

    const id = JSON.parse(res.body)["id"]

    for (let j = 0; j < bindingsPerUser; j++) {
        const res = retryOnConflict(() => {
            return http.post(
                `${baseUrl}/v3/globalrolebindings`,
                JSON.stringify({
                    "type": "globalRoleBinding",
                    "globalRoleId": [bindings[(i * bindingsPerUser + j) % bindings.length]],
                    "userId": id
                }),
                {cookies: cookies}
            )
        })

        check(res, {
            '/v3/globalrolebindings returns status 201': (r) => r.status === 201 || r.status === 204,
        })
    }

    resourceMetric.add(roleCount + userCount)
}
