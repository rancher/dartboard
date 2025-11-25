import encoding from 'k6/encoding';
import exec from 'k6/execution';
import { createSecretsWithLabels, createStorageClasses } from '../generic/generic_utils.js';
import { login, getCookies } from '../rancher/rancher_utils.js';
import {fail} from 'k6';
import {loadKubeconfig} from '../generic/k8s.js'
import * as k8s from '../generic/k8s.js'
import { customHandleSummary } from '../generic/k6_utils.js';

// Parameters
const namespace = "longhorn-system"
const token = __ENV.TOKEN
const secretCount = Number(__ENV.SECRET_COUNT)
const secretData = encoding.b64encode("a".repeat(10*1024))
const key = __ENV.KEY || "blue"
const value = __ENV.VALUE || "green"
const clusterId = __ENV.CLUSTER || "local"
const vus = __ENV.VUS || 2

// Option setting
const kubeconfig = loadKubeconfig(__ENV.KUBECONFIG, __ENV.CONTEXT)
const baseUrl = kubeconfig["url"].replace(":6443", "")
const username = __ENV.USERNAME
const password = __ENV.PASSWORD

export const handleSummary = customHandleSummary;

export const options = {
    insecureSkipTLSVerify: true,
    tlsAuth: [
        {
            cert: kubeconfig["cert"],
            key: kubeconfig["key"],
        },
    ],

    setupTimeout: '8h',

    scenarios: {
        createResourceSecretsWithLabels: {
            executor: 'shared-iterations',
            exec: 'createResourceSecretsWithLabels',
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
  // if session cookie was specified, save it
  if (token) {
    return { R_SESS: token }
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

    pause()

    const cookies = http.cookieJar().cookiesForURL(res.url)

    return cookies
  }

  return {}
}

// create storage classes
export function createResourceSecretsWithLabels(cookies) {
    const iter = exec.scenario.iterationInTest
    createSecretsWithLabels(baseUrl, cookies, secretData, clusterId, namespace, iter, key, value)
}