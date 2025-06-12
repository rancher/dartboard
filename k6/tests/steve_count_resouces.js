import { check, fail } from 'k6';
import http from 'k6/http';

// Continuously prints count of a resource on the console

// Parameters
const namespace = __ENV.NAMESPACE || "scalability-test"
const vus = __ENV.VUS || 1
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD
const token = __ENV.TOKEN
const cluster = __ENV.CLUSTER || "local"
const resource = __ENV.RESOURCE || "pods"
const rate =  __ENV.RATE || 2
const duration = '2h'

// Option setting
export const options = {
    insecureSkipTLSVerify: true,

    scenarios: {
        list : {
            executor: 'constant-arrival-rate',
            exec: 'count',
            preAllocatedVUs: vus,
            duration: duration,
            rate: rate,
        }
    },
    thresholds: {
        checks: ['rate>0.99']
    }
}

// Test functions, in order of execution

export function setup() {
    // if session cookie was specified, save it
    if (token) {
        return {R_SESS: token}
    }

    // if credentials were specified, log in
    if (username && password) {
        const res = http.post(`${baseUrl}/v3-public/localProviders/local?action=login`, JSON.stringify({
            "description": "UI session",
            "responseType": "cookie",
            "username": username,
            "password": password
        }))

        check(res, {
            'logging in returns status 200': (r) => r.status === 200,
        })

        return http.cookieJar().cookiesForURL(res.url)
    }

    return {}
}

export function count(cookies) {
    const url = cluster === "local"?
        `${baseUrl}/v1/${resource}` :
        `${baseUrl}/k8s/clusters/${cluster}/v1/${resource}`

    const fullUrl = url + `?projectsornamespaces=${namespace}&page=1&pageSize=1`
    const res = http.get(fullUrl, {cookies: cookies})

    const criteria = {}
    criteria[`listing ${resource} from cluster ${cluster} succeeds`] = (r) => r.status === 200
    criteria[`no slow pagination errors (410 Gone) detected`] = (r) => r.status !== 410
    check(res, criteria)

    try {
        const body = JSON.parse(res.body)
        console.log(`${resource} count: ${body.count}`)
    } catch (e) {
        if (e instanceof SyntaxError) {
            fail("Response body does not parse as JSON: " + res.body)
        }
        throw e
    }
}
