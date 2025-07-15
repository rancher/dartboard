import {check, fail, sleep} from 'k6';
import exec from 'k6/execution';
import http from 'k6/http';
import {Gauge} from 'k6/metrics';
import * as projectUtil from "../projects/project_utils.js";
import {getCookies, login} from "../rancher/rancher_utils.js";
import {getPrincipalIds, getCurrentUserPrincipalId, getClusterIds} from "../rancher/rancher_users_utils.js"

// Parameters
const projectCount = Number(__ENV.PROJECT_COUNT)
const vus = 1
const customRoleTemplateBindingsPerProject = 5

// Option setting
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD

// Option setting
export const options = {
    insecureSkipTLSVerify: true,

    setupTimeout: '8h',

    scenarios: {
        createProjects: {
            executor: 'shared-iterations',
            exec: 'createProjects',
            vus: vus,
            iterations: projectCount,
            maxDuration: '1h',
        },
    },
    thresholds: {
        checks: ['rate>0.99']
    }
}

// Custom metrics
const projectsMetric = new Gauge('test_projects')

// Test functions, in order of execution

export function setup() {
    // log in
    if (!login(baseUrl, {}, username, password).status === 200) {
        fail(`could not login into cluster`)
    }
    const cookies = getCookies(baseUrl)

    // delete leftovers, if any
    cleanup(cookies)
    // return data that remains constant throughout the test
    return {
        cookies: cookies,
        principalIds: getPrincipalIds(baseUrl, cookies),
        myId: getCurrentUserPrincipalId(baseUrl, cookies),
        clusterIds: getClusterIds(baseUrl, cookies)
    }
}

function cleanup(cookies) {
    let res = http.get(`${baseUrl}/v1/management.cattle.io.projects`, {cookies: cookies})
    check(res, {
        '/v1/management.cattle.io.projects returns status 200': (r) => r.status === 200,
    })
    let { _, projectArray } = projectUtil.getNormanProjectsMatchingName(baseUrl, cookies, "Test ")
      console.log(`Found ${projectArray.length} projects to clean up`)
      projectArray.forEach(r => {
        let delRes = projectUtil.deleteNormanProject(baseUrl, cookies, r["id"])
        if (delRes.status !== 200 && delRes.status !== 204) deleteAllFailed = true
        sleep(0.5)
      })
}

const mainRoleTemplateIds = ["project-owner", "project-member", "read-only", "custom"]
const customRoleTemplateIds = ["create-ns", "configmaps-manage", "ingress-manage", "projectcatalogs-manage", "projectroletemplatebindings-manage", "secrets-manage", "serviceaccounts-manage", "services-manage", "persistentvolumeclaims-manage", "workloads-manage", "configmaps-view", "ingress-view", "monitoring-ui-view", "projectcatalogs-view", "projectroletemplatebindings-view", "secrets-view", "serviceaccounts-view", "services-view", "persistentvolumeclaims-view", "workloads-view"]

export function createProjects(data) {
    let response
    const i = exec.scenario.iterationInTest
    const cookies = data.cookies
    const myId = data.myId
    const clusterId = data.clusterIds[i % data.clusterIds.length]

    response = http.post(
        `${baseUrl}/v3/projects`,
        JSON.stringify({
            "type": "project",
            "name": `Test Project ${i}`,
            "description": `Test Project ${i}`,
            "annotations": {},
            "labels": {},
            "clusterId": clusterId,
            "creatorId": `local://${myId}`,
            "containerDefaultResourceLimit": {
                "limitsCpu": "4m",
                "limitsMemory": "5Mi",
                "requestsCpu": "2m",
                "limitsGpu": 6,
                "requestsMemory": "3Mi"
            },
            "resourceQuota": {
                "limit": {
                    "configMaps": "9",
                    "limitsMemory": "900Mi",
                    "limitsCpu": "90m",
                    "persistentVolumeClaims": "9000"
                }
            },
            "namespaceDefaultResourceQuota": {
                "limit": {
                    "configMaps": "6",
                    "limitsMemory": "600Mi",
                    "limitsCpu": "60m",
                    "persistentVolumeClaims": "6000"
                }
            }
        }),
        { cookies: cookies }
    )
    check(response, {
        '/v3/projects returns status 201': (r) => r.status === 201,
    })

    const projectId = JSON.parse(response.body)["id"]

    response = http.post(
        `${baseUrl}/v3/projects/${projectId.replace("/", ":")}?action=setpodsecuritypolicytemplate`,
        JSON.stringify({"podSecurityPolicyTemplateId": null}),
        { cookies: cookies }
    )
    check(response, {
        'setpodsecuritypolicytemplate works': (r) => r.status === 200 || r.status === 409,
    })

    const principalId = data.principalIds[i % data.principalIds.length]
    const mainRoleTemplateId = mainRoleTemplateIds[i % mainRoleTemplateIds.length]
    const roleTemplateIds = mainRoleTemplateId !== "custom" ? [mainRoleTemplateId] : Array.from({length: customRoleTemplateBindingsPerProject}, (_, j) => (
        customRoleTemplateIds[i % customRoleTemplateIds.length]
    ))

    for (const roleTemplateId of roleTemplateIds) {

        // HACK: creating projectroletemplatebindings might fail with 404 if the project controller is too slow
        // allow up to 30 retries
        let success = false
        for (let j = 0; j < 30 && !success; j++) {
            response = http.post(
                `${baseUrl}/v3/projectroletemplatebindings`,
                JSON.stringify({
                    "type": "projectroletemplatebinding",
                    "roleTemplateId": roleTemplateId,
                    "userPrincipalId": `local://${principalId}`,
                    "projectId": projectId
                }),
                { cookies: cookies }
            )
            check(response, {
                '/v3/projectroletemplatebindings returns status 201 or 404': (r) => r.status === 201 || r.status === 404,
            })

            success = response.status === 201
            if (!success) {
                sleep(Math.random())
            }
        }
        if (!success) {
            fail("/v3/projectroletemplatebindings did not return 201 after 30 attempts")
        }
    }

    projectsMetric.add(projectCount)
}
