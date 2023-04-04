import { check } from 'k6';
import http from 'k6/http';
import { textSummary } from './lib/k6-summary-0.0.2.js';

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

    let revision = null
    let continueToken = null
    while (true) {
        const fullUrl = url + "?limit=100" +
            (revision != null ? "&revision=" + revision : "") +
            (continueToken != null ? "&continue=" + continueToken : "")

        const res = http.get(fullUrl, {cookies: cookies})

        check(res, {
            '/v1/configmaps returns status 200': (r) => r.status === 200,
        })

        const body = JSON.parse(res.body)
        if (body === undefined || body.continue === undefined) {
            break
        }
        if (revision == null) {
            revision = body.revision
        }
        continueToken = body.continue
    }
}

export function handleSummary(data) {
    return {
        'stdout': textSummary(data, { indent: ' ', enableColors: true }), // keep default text output
        "out.json": JSON.stringify(data), // additionally dump raw data in JSON form for later processing
    }
}
