import encoding from 'k6/encoding';
import exec from 'k6/execution';
import { createConfigMaps, createSecrets, createDeployments } from '../generic/generic_utils.js';
import {check, fail, sleep} from 'k6';
import * as k8s from '../generic/k8s.js'
import http from 'k6/http';


// Parameters
const namespace = "scalability-test"
const secretData = encoding.b64encode("a".repeat(10*1024))
const confgMapData = encoding.b64encode("a".repeat(10*1024))
const secretCount = Number(__ENV.SECRET_COUNT)
const configMapCount = Number(__ENV.CONFIGMAP_COUNT)
const deploymentCount =Number(__ENV.DEPLOYMENT_COUNT)
const vus = 10

// Option setting
const kubeconfig = k8s.kubeconfig(__ENV.KUBECONFIG, __ENV.CONTEXT)
const baseUrl = kubeconfig["url"].replace(":6443", "")
const username = __ENV.USERNAME
const password = __ENV.PASSWORD

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
        createVaiResourcesSecrets: {
            executor: 'shared-iterations',
            exec: 'createVaiResourcesSecrets',
            vus: vus,
            iterations: secretCount,
            maxDuration: '1h',
        },
        createVaiResourcesConfigMaps: {
            executor: 'shared-iterations',
            exec: 'createVaiResourcesConfigMaps',
            vus: vus,
            iterations: configMapCount,
            maxDuration: '1h',
        },
        createVaiResourcesDeployments: {
            executor: 'shared-iterations',
            exec: 'createVaiResourcesDeployments',
            vus: vus,
            iterations: deploymentCount,
            maxDuration: '1h',
        },
    },
    thresholds: {
        checks: ['rate>0.99']
    }
};

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

    const cookies = http.cookieJar().cookiesForURL(res.url)

    // delete leftovers, if any
    k8s.del(`${baseUrl}/api/v1/namespaces/${namespace}`)

    // create empty namespace
    const body = {
        "metadata": {
            "name": namespace,
            
        },
    }
    k8s.create(`${baseUrl}/api/v1/namespaces`, body)

    return cookies
}

// create secrets
export function createVaiResourcesSecrets(cookies) {
    const iter = exec.scenario.iterationInTest
    createSecrets(iter, namespace, baseUrl, secretData, cookies)
}

// create config maps
export function createVaiResourcesConfigMaps(cookies) {
    const iter = exec.scenario.iterationInTest
    createConfigMaps(iter, namespace, baseUrl, confgMapData, cookies)
}

// create deployments
export function createVaiResourcesDeployments(cookies) {
    const iter = exec.scenario.iterationInTest
    createDeployments(iter, namespace, baseUrl, cookies)
}
