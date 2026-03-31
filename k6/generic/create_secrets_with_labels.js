import encoding from 'k6/encoding';
import exec from 'k6/execution';
import { createSecretWithLabel } from '../generic/generic_utils.js';
import * as k8s from '../generic/k8s.js'
import { customHandleSummary } from '../generic/k6_utils.js';

// Parameters
const namespace = __ENV.NAMESPACE || "scalability-test"
const token = __ENV.TOKEN
const secretCount = Number(__ENV.SECRET_COUNT)
const secretData = encoding.b64encode("a".repeat(10*1024))
const key = __ENV.KEY || "blue"
const value = __ENV.VALUE || "green"
const clusterId = __ENV.CLUSTER || "local"
const vus = __ENV.VUS || 2

// Option setting
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD

export const handleSummary = customHandleSummary;

export const options = {
    insecureSkipTLSVerify: true,
    tlsAuth: [
        {
            cert: k8s.kubeconfig["cert"],
            key: k8s.kubeconfig["key"],
        },
    ],

    setupTimeout: '8h',

    scenarios: {
        createResourceSecretWithLabel: {
            executor: 'shared-iterations',
            exec: 'createResourceSecretWithLabel',
            vus: vus,
            iterations: secretCount,
            maxDuration: '1h',
        },
    },
    thresholds: {
        http_req_failed: ['rate<=0.01'], // http errors should be less than 1%
        http_req_duration: ['p(99)<=500'], // 99% of requests should be below 500ms
        checks: ['rate>0.99'], // the rate of successful checks should be higher than 99%
    }
};

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

// create storage classes
export function createResourceSecretWithLabel(cookies) {
    const iter = exec.scenario.iterationInTest
    createSecretWithLabel(baseUrl, cookies, secretData, clusterId, namespace, iter, key, value)
}