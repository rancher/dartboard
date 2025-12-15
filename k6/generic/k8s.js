import { check, sleep } from 'k6';
import encoding from 'k6/encoding';
import http from 'k6/http';
import { WebSocket } from 'k6/experimental/websockets';
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

    const result = {
        url: cluster['server']
    };
    console.debug(`Kubernetes API server URL: ${result.url}`);

    if (user['client-certificate-data'] !== undefined) {
        result.cert = encoding.b64decode(user['client-certificate-data'], 'std', 's');
    } else {
        console.debug("Could not find client-certificate-data in kubeconfig.");
    }
    if (user['client-key-data'] !== undefined) {
        result.key = encoding.b64decode(user['client-key-data'], 'std', 's');
    } else {
        console.debug("Could not find client-key-data in kubeconfig.");
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

// executes a command in a pod and returns the output
export function exec(baseUrl, namespace, podName, container, command, timeoutSeconds = 30) {
    const params = new URLSearchParams();
    params.append('container', container);
    params.append('stdout', 'true');
    params.append('stderr', 'true');
    
    if (Array.isArray(command)) {
        command.forEach(cmd => params.append('command', cmd));
    } else {
        params.append('command', 'sh');
        params.append('command', '-c');
        params.append('command', command);
    }

    const wsUrl = `${baseUrl}/api/v1/namespaces/${namespace}/pods/${podName}/exec?${params.toString()}`
        .replace('https://', 'wss://')
        .replace('http://', 'ws://');

    let output = '';
    let errorOutput = '';
    let completed = false;

    const ws = new WebSocket(wsUrl, 'v4.channel.k8s.io');

    ws.addEventListener('open', () => {
        console.debug(`Exec WebSocket connected to ${podName}`);
    });

    ws.addEventListener('message', (e) => {
        if (typeof e.data === 'string') {
            const data = e.data;
            if (data.length > 1) {
                const channel = data.charCodeAt(0);
                const content = data.substring(1);
                
                if (channel === 1) {
                    output += content;
                } else if (channel === 2) {
                    errorOutput += content;
                }
            }
        }
    });

    ws.addEventListener('close', () => {
        completed = true;
        console.debug(`Exec WebSocket closed for ${podName}`);
    });

    ws.addEventListener('error', (e) => {
        console.error(`Exec WebSocket error: ${e.error}`);
        completed = true;
    });

    const startTime = Date.now();
    while (!completed && (Date.now() - startTime) < timeoutSeconds * 1000) {
        sleep(0.1);
    }

    if (!completed) {
        ws.close();
    }

    return {
        stdout: output.trim(),
        stderr: errorOutput.trim(),
        success: errorOutput === ''
    };
}
