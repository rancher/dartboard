import { check, sleep } from 'k6';
import encoding from 'k6/encoding';
import http from 'k6/http';
import * as YAML from '../lib/js-yaml-4.1.0.mjs'

import { URL } from '../lib/url-1.0.0.js';

const timeout = '3600s'

function loadKubeconfig(file, contextName) {
    const config = YAML.load(open(file));

    console.debug(`Loading kubeconfig from '${file}' using context '${contextName}'.`);

    const context = config["contexts"].find(c => c["name"] === contextName)["context"]
    const clusterName = context["cluster"]
    const userName = context["user"]
    console.debug(`Found context: cluster='${clusterName}', user='${userName}'.`);

    const cluster = config["clusters"].find(c => c["name"] === clusterName)["cluster"]
    const user = config["users"].find(c => c["name"] === userName)["user"]

    if (user['client-certificate-data']) {
        console.debug("Found client-certificate-data in kubeconfig.");
    } else {
        console.debug("Could not find client-certificate-data in kubeconfig.");
    }
    if (user['client-key-data']) {
        console.debug("Found client-key-data in kubeconfig.");
    } else {
        console.debug("Could not find client-key-data in kubeconfig.");
    }

    const result = {
        url: cluster['server']
    };
    console.debug(`Kubernetes API server URL: ${result.url}`);

    if (user['client-certificate-data']) {
        result.cert = encoding.b64decode(user['client-certificate-data'], 'std', 's');
    }
    if (user['client-key-data']) {
        result.key = encoding.b64decode(user['client-key-data'], 'std', 's');
    }
    if (user['token']) {
        result.token = user.token;
    }

    return result;
}

export const kubeconfig = loadKubeconfig(__ENV.KUBECONFIG, __ENV.CONTEXT);

// creates a k8s resource
export function create(url, body, retry = true){
    const res = http.post(url, JSON.stringify(body));

    const success = check(res, {
        'POST returns status 201 or 409': (r) => r.status === 201 || r.status === 409,
    })

    if (!success) {
        console.error(`POST to ${url} failed. Status: ${res.status}, Body: ${res.body}, Request Headers: ${JSON.stringify(res.request.headers)}`);
    }

    if (res.status === 409 && retry) {
        // wait a bit and try again
        sleep(Math.random())

        return create(url, body)
    }

    return res
}

// deletes a k8s resource
export function del(url){
    const res = http.del(url)

    const success = check(res, {
        'DELETE returns status 200 or 404': (r) => r.status === 200 || r.status === 404,
    })

    if (!success) {
        console.error(`DELETE to ${url} failed. Status: ${res.status}, Body: ${res.body}, Request Headers: ${JSON.stringify(res.request.headers)}`);
    }

    return res
}

const continueRegex = /"continue":"([A-Za-z0-9]+)"/;

// lists k8s resources
export function list(url, limit) {
    let _continue = 'first'
    let responses = []

    while (_continue != null) {
        const fullUrl = new URL(url);
        fullUrl.searchParams.append('limit', limit);
        fullUrl.searchParams.append('timeout', timeout);
        fullUrl.searchParams.append('watch', false);
        if (_continue !== 'first') {
            fullUrl.searchParams.append('continue', _continue);
        }

        const res = http.get(fullUrl.toString());

        const success = check(res, {
            'list returns status 200': (r) => r.status === 200,
        });

        if (!success) {
            console.error(`GET to ${fullUrl.toString()} failed. Status: ${res.status}, Body: ${res.body}, Request Headers: ${JSON.stringify(res.request.headers)}`);
        }

        const found = res.body.match(continueRegex);
        _continue = found !== null ? found[1] : null

        responses.push(res)
    }

    return responses
}
