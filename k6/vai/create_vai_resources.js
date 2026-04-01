import encoding from 'k6/encoding';
import exec from 'k6/execution';
import { createConfigMap, createSecret, createDeployment } from '../generic/generic_utils.js';
import { login, getCookies } from '../rancher/rancher_utils.js';
import {fail} from 'k6';
import * as k8s from '../generic/k8s.js'
import { customHandleSummary } from '../generic/k6_utils.js';

// Parameters
const token = __ENV.TOKEN
const namespace = "scalability-test"
const secretData = encoding.b64encode("a".repeat(10*1024))
const configMapData = encoding.b64encode("a".repeat(10*1024))
const secretCount = Number(__ENV.SECRET_COUNT)
const configMapCount = Number(__ENV.CONFIGMAP_COUNT)
const deploymentCount =Number(__ENV.DEPLOYMENT_COUNT)
const clusterId = __ENV.CLUSTER || "local"
const vus = 5

// Option setting
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD

export const handleSummary = customHandleSummary;

export const options = {
    insecureSkipTLSVerify: true,

    setupTimeout: '8h',

    scenarios: {
        createVaiResourcesSecrets: {
            executor: 'shared-iterations',
            exec: 'createVaiResourceSecret',
            vus: vus,
            iterations: secretCount,
            maxDuration: '1h',
        },
        createVaiResourcesConfigMaps: {
            executor: 'shared-iterations',
            exec: 'createVaiResourceConfigMap',
            vus: vus,
            iterations: configMapCount,
            maxDuration: '1h',
        },
        createVaiResourcesDeployments: {
            executor: 'shared-iterations',
            exec: 'createVaiResourceDeployment',
            vus: vus,
            iterations: deploymentCount,
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

       // create empty namespace
    const body = {
        "metadata": {
            "name": namespace,

        },
    }

    const create_url = clusterId === "local"?
        `${baseUrl}/v1/namespaces` :
        `${baseUrl}/k8s/clusters/${clusterId}/v1/namespaces`

    k8s.create(`${create_url}`, body)
    

  return {}
}

// create secrets
export function createVaiResourceSecret(cookies) {
    const iter = exec.scenario.iterationInTest
    createSecret(baseUrl, cookies, secretData, clusterId, namespace, iter)
}

// create config maps
export function createVaiResourceConfigMap(cookies) {
    const iter = exec.scenario.iterationInTest
    createConfigMap(baseUrl, cookies, configMapData, clusterId, namespace, iter)
}

// create deployments
export function createVaiResourceDeployment(cookies) {
    const iter = exec.scenario.iterationInTest
    createDeployment(baseUrl, cookies, clusterId, namespace, iter)
}
