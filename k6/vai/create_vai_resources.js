import encoding from 'k6/encoding';
import exec from 'k6/execution';
import { createConfigMaps, createSecrets, createDeployments } from '../generic/generic_utils.js';
import { login, getCookies } from '../rancher/rancher_utils.js';
import {fail} from 'k6';
import * as k8s from '../generic/k8s.js'


// Parameters
const namespace = "scalability-test"
const secretData = encoding.b64encode("a".repeat(10*1024))
const configMapData = encoding.b64encode("a".repeat(10*1024))
const secretCount = Number(__ENV.SECRET_COUNT)
const configMapCount = Number(__ENV.CONFIGMAP_COUNT)
const deploymentCount =Number(__ENV.DEPLOYMENT_COUNT)
const clusterId = __ENV.CLUSTER || "local"
const vus = 5

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
        http_req_failed: ['rate<=0.01'], // http errors should be less than 1%
        http_req_duration: ['p(99)<=500'], // 99% of requests should be below 500ms
        checks: ['rate>0.99'], // the rate of successful checks should be higher than 99%
    }
};

export function setup() {
    
    // log in
    if (!login(baseUrl, {}, username, password)) {
        fail(`could not login into cluster`)
    }
    const cookies = getCookies(baseUrl)

    const del_url = clusterId === "local"?
        `${baseUrl}/v1/namespaces/${namespace}` :
        `${baseUrl}/k8s/clusters/${clusterId}/v1/namespaces/${namespace}`

    // delete leftovers, if any
    k8s.del(`${del_url}`)

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

    return cookies
}

// create secrets
export function createVaiResourcesSecrets(cookies) {
    const iter = exec.scenario.iterationInTest
    createSecrets(baseUrl, cookies, secretData, clusterId, namespace, iter)
}

// create config maps
export function createVaiResourcesConfigMaps(cookies) {
    const iter = exec.scenario.iterationInTest
    createConfigMaps(baseUrl, cookies, configMapData, clusterId, namespace, iter)
}

// create deployments
export function createVaiResourcesDeployments(cookies) {
    const iter = exec.scenario.iterationInTest
    createDeployments(baseUrl, cookies, clusterId, namespace, iter)
}
