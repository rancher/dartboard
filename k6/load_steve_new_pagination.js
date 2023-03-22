import { check } from 'k6';
import http from 'k6/http';

// Parameters
const vus = __ENV.VUS
const perVuIterations = __ENV.PER_VU_ITERATIONS
const baseUrl = __ENV.BASEURL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD
const cluster = __ENV.CLUSTER

// Option setting
export const options = {
    insecureSkipTLSVerify: true,

    summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(95)', 'p(99)', 'count'],

    scenarios: {
        list : {
            executor: 'per-vu-iterations',
            exec: 'list',
            vus: vus,
            iterations: perVuIterations,
            maxDuration: '24h',
        }
    },
    thresholds: {
        checks: ['rate>0.99']
    }
};


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

    return http.cookieJar().cookiesForURL(res.url)
}

export function list(cookies) {
    const url = cluster === "local"?
        `${baseUrl}/v1/configmaps` :
        `${baseUrl}/k8s/clusters/${cluster}/v1/configmaps`

    let i = 1
    let revision = null
    while (true) {
        const fullUrl = url + "?pagesize=100&page=" + i +
            (revision != null ? "&revision=" + revision : "")

        const res = http.get(fullUrl, {cookies: cookies})

        check(res, {
            '/v1/configmaps returns status 200': (r) => r.status === 200,
        })

        const body = JSON.parse(res.body)
        if (body === undefined || body.data === undefined || body.data.length === 0) {
            break
        }
        if (revision == null) {
            revision = body.revision
        }
        i = i + 1
    }
}
