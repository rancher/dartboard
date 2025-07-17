import encoding from 'k6/encoding';
import exec from 'k6/execution';
import { check, fail } from 'k6';
import * as k8s from './k8s.js'
import http from 'k6/http';

// Parameters
const count = Number(__ENV.COUNT)
const vus = 10

// Option setting
const kubeconfig = k8s.kubeconfig(__ENV.KUBECONFIG, __ENV.CONTEXT)
const baseUrl = kubeconfig["url"].replace(":6443", "")

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
        createStorageClasses: {
            executor: 'shared-iterations',
            exec: 'createStorageClasses',
            vus: vus,
            iterations: count,
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
    cleanup(cookies)

    return cookies
}


function cleanup(cookies) {
    let res = http.get(`${baseUrl}/v1/storage.k8s.io.storageclasses`, {cookies: cookies})
    check(res, {
        '/v1/storage.k8s.io.storageclasses returns status 200': (r) => r.status === 200,
    })
    JSON.parse(res.body)["data"].filter(r => r["id"].startsWith("test-")).forEach(r => {
        res = http.del(`${baseUrl}/v1/storage.k8s.io.storageclasses/`+`${r["id"]}`, {cookies: cookies})
        check(res, {
            'DELETE /v1/storage.k8s.io.storageclasses returns status 204': (r) => r.status === 204,
        })
    })
}

export function createStorageClasses(cookies){

    const name = `test-storage-class-${exec.scenario.iterationInTest}`

    const create = http.post(`${baseUrl}/v1/storage.k8s.io.storageclasses`, JSON.stringify({
        "type": "storage.k8s.io.storageclass",
        "metadata": {
            "name": name
        },
        "provisioner": "driver.longhorn.io",
        }),
        {cookies: cookies})

    check(create, {
        '/v1/storage.k8s.io.storageclasses returns status 201': (r) => r.status === 201,
    })

 }
