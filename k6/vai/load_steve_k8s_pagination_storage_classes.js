import { check, fail, sleep } from 'k6';
import * as k8s from '../generic/k8s.js'
import { login, getCookies } from '../rancher/rancher_utils.js';
import http from 'k6/http';
import { customHandleSummary } from '../generic/k6_utils.js';


// Parameters
const vus = __ENV.VUS
const perVuIterations = __ENV.PER_VU_ITERATIONS
const kubeconfig = k8s.kubeconfig(__ENV.KUBECONFIG, __ENV.CONTEXT)
const baseUrl = kubeconfig["url"].replace(":6443", "")
const username = __ENV.USERNAME
const password = __ENV.PASSWORD
const clusterId = __ENV.CLUSTER || "local"
const paginationStyle = __ENV.PAGINATION_STYLE || "k8s"
const pauseSeconds = parseFloat(__ENV.PAUSE_SECONDS || 0.0)

export const handleSummary = customHandleSummary;

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
}

// Simulate a pause after a click - on average pauseSeconds, +/- a random quantity up to 50%
function pause() {
  sleep(pauseSeconds + (Math.random() - 0.5) * 2 * pauseSeconds / 2)
}

export function setup() {
    if (!login(baseUrl, {}, username, password)) {
        fail(`could not login into cluster`)
    }
    const cookies = getCookies(baseUrl)

    return cookies
}

export function list(cookies, filters = "") {
    if (paginationStyle === "k8s") {
      listStorageClasses(cookies)
    }
    else if (paginationStyle === "steve") {
      listStorageClassesVai(cookies)
    }
    else {
      fail("Invalid PAGINATION_STYLE value: " + paginationStyle)
    }
}

export function listStorageClasses(cookies) {
    const url = clusterId === "local"?
        `${baseUrl}/v1/storage.k8s.io.storageclasses` :
        `${baseUrl}/k8s/clusters/${clusterId}/v1/storage.k8s.io.storageclasses`

    let revision = null
    let continueToken = null
    while (true) {
        const fullUrl = url + "?limit=100" +
            (revision != null ? "&revision=" + revision : "") +
            (continueToken != null ? "&continue=" + continueToken : "")

        const res = http.get(fullUrl, {cookies: cookies})

        check(res, {
            '/v1/storage.k8s.io.storageclasses returns status 200': (r) => r.status === 200,
        })

        try {
            const body = JSON.parse(res.body)
            if (body === undefined || body.continue === undefined) {
                break
            }
            if (revision == null) {
                revision = body.revision
            }
            continueToken = body.continue
        }
        catch (e){
            if (e instanceof SyntaxError) {
                fail("Response body does not parse as JSON: " + res.body)
            }
            throw e
        }
    }

    pause()
}

export function listStorageClassesVai(cookies) {
    const url = clusterId === "local"?
        `${baseUrl}/v1/storage.k8s.io.storageclasses` :
        `${baseUrl}/k8s/clusters/${clusterId}/v1/storage.k8s.io.storageclasses`

    let i = 1
    let revision = null
    while (true) {
        const fullUrl = url + "?pagesize=100&page=" + i +
            (revision != null ? "&revision=" + revision : "")

        const res = http.get(fullUrl, {cookies: cookies})

        check(res, {
            '/v1/storage.k8s.io.storageclasses returns status 200': (r) => r.status === 200,
        })

        try {
            const body = JSON.parse(res.body)
            if (body === undefined || body.data === undefined || body.data.length === 0) {
                break
            }
            if (revision == null) {
                revision = body.revision
            }
            i = i + 1
        }
        catch (e){
            if (e instanceof SyntaxError) {
                fail("Response body does not parse as JSON: " + res.body)
            }
            throw e
        }
    }

    pause()
}

