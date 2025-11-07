import encoding from 'k6/encoding';
import exec from 'k6/execution';
import { createStorageClasses } from '../generic/generic_utils.js';
import { login, getCookies } from '../rancher/rancher_utils.js';
import {fail} from 'k6';
import * as k8s from '../generic/k8s.js'
import { customHandleSummary } from '../generic/k6_utils.js';

// Parameters
const namespace = "scalability-test"
const storageClassCount =Number(__ENV.STORAGECLASS_COUNT)
const clusterId = "local"
const vus = __ENV.VUS || 2

// Option setting
const kubeconfig = k8s.kubeconfig(__ENV.KUBECONFIG, __ENV.CONTEXT)
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
        createResourcesStorageClasses: {
            executor: 'shared-iterations',
            exec: 'createResourcesStorageClasses',
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

    // log in
    if (!login(baseUrl, {}, username, password)) {
        fail(`could not login into cluster`)
    }
    const cookies = getCookies(baseUrl)

    return cookies
}

// create storage classes
export function createResourcesStorageClasses(cookies) {
    const iter = exec.scenario.iterationInTest
    createStorageClasses(baseUrl, cookies, clusterId, iter)
}
