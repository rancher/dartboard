import { check } from 'k6';
import { sleep } from 'k6';
import encoding from 'k6/encoding';
import http from 'k6/http';
import * as YAML from './lib/js-yaml-4.1.0.mjs'

import { URL } from './lib/url-1.0.0.js';

const limit = 5000
const timeout = '3600s'

// loads connection variables from kubeconfig's specified context
export function kubeconfig(file, contextName){
    const config = YAML.load(open(file));

    const context = config["contexts"].find(c => c["name"] === contextName)["context"]
    const clusterName = context["cluster"]
    const userName = context["user"]

    const cluster = config["clusters"].find(c => c["name"] === clusterName)["cluster"]
    const user = config["users"].find(c => c["name"] === userName)["user"]

    return {
        url : cluster['server'],
        cert : encoding.b64decode(user['client-certificate-data'], 'std', 's'),
        key: encoding.b64decode(user['client-key-data'], 'std', 's')
    }
}

// creates a k8s resource
export function create(url, body, retry = true){
    const res = http.post(url, JSON.stringify(body));

    check(res, {
        'POST returns status 201 or 409': (r) => r.status === 201 || r.status === 409,
    })

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

    check(res, {
        'DELETE returns status 200 or 404': (r) => r.status === 200 || r.status === 404,
    })

    return res
}

const continueRegex = /"continue":"([A-Za-z0-9]+)"/;

// lists k8s resources
export function list(url) {
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

        check(res, {
            'list returns status 200': (r) => r.status === 200,
        });

        const found = res.body.match(continueRegex);
        _continue = found !== null ? found[1] : null

        responses.push(res)
    }

    return responses
}
