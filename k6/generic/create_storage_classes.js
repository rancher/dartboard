import exec from 'k6/execution';
import * as k8s from '../generic/k8s.js'
import { customHandleSummary } from '../generic/k6_utils.js';
import { createStorageClass } from '../generic/generic_utils.js';

// Parameters
const token = __ENV.TOKEN
const storageClassCount =Number(__ENV.STORAGECLASS_COUNT)
const clusterId = "local"
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
        createResourceStorageClass: {
            executor: 'shared-iterations',
            exec: 'createResourceStorageClass',
            vus: vus,
            iterations: storageClassCount,
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
export function createResourceStorageClass(cookies) {
    const iter = exec.scenario.iterationInTest
    createStorageClass(baseUrl, cookies, clusterId, iter)
}
