import { check, fail, sleep } from 'k6';
import * as k8s from '../generic/k8s.js'
import { login, getCookies } from '../rancher/rancher_utils.js';
import http from 'k6/http';
import { customHandleSummary } from '../generic/k6_utils.js';

// Parameters
const vus = __ENV.VUS
const perVuIterations = __ENV.PER_VU_ITERATIONS
const token = __ENV.TOKEN
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD
const clusterId = __ENV.CLUSTER || "local"
const paginationStyle = __ENV.PAGINATION_STYLE || "k8s"
const pauseSeconds = parseFloat(__ENV.PAUSE_SECONDS || 0.0)

const key = __ENV.KEY || "blue"
const value = __ENV.VALUE || "green"

export const handleSummary = customHandleSummary;

// Option setting
export const options = {
    insecureSkipTLSVerify: true,
    tlsAuth: [
        {
            cert: k8s.kubeconfig["cert"],
            key: k8s.kubeconfig["key"],
        },
    ],

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
  var cookies = {}

  if (token) {
    cookies = { R_SESS: token }
  } else if (username != "" && password != "") {

    let loginRes = login(baseUrl, {}, username, password)

    if (loginRes.status !== 200) {
      fail(`could not login to cluster`)
    }
    cookies = getCookies(baseUrl)
  } else {
    fail("Must provide token or login credentials")
  }

  return cookies
}

export function list(cookies, filters = "") {
    if (paginationStyle === "k8s") {
      listFilterSort(cookies)
    }
    else if (paginationStyle === "steve") {
      listFilterSortVai(cookies)
    }
    else {
      fail("Invalid PAGINATION_STYLE value: " + paginationStyle)
    }
}

export function listFilterSort(cookies) {
    const url = clusterId === "local"?
        `${baseUrl}/v1/secrets` :
        `${baseUrl}/k8s/clusters/${clusterId}/v1/secrets`

    let revision = null
    let continueToken = null
    while (true) {
        const fullUrl = url + "?limit=100" + "&sort=metadata.name&filter=metadata.labels." + key + "=" + value +
            (revision != null ? "&revision=" + revision : "") +
            (continueToken != null ? "&continue=" + continueToken : "")

        const res = http.get(fullUrl, {cookies: cookies})

        check(res, {
            '/v1/secrets returns status 200': (r) => r.status === 200,
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


export function listFilterSortVai(cookies) {
    const url = clusterId === "local"?
        `${baseUrl}/v1/secrets` :
        `${baseUrl}/k8s/clusters/${clusterId}/v1/secrets`

    let i = 1
    let revision = null
    while (true) {
        const fullUrl = url + "?pagesize=100&page=" + i + "&sort=metadata.name&filter=metadata.labels." + key + "=" + value +
            (revision != null ? "&revision=" + revision : "")

        const res = http.get(fullUrl, {cookies: cookies})

        check(res, {
            '/v1/secrets returns status 200': (r) => r.status === 200,
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
