import { check, fail, sleep } from 'k6';
import http from 'k6/http';
import { Trend } from 'k6/metrics';
import { WebSocket } from 'k6/experimental/websockets';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';
import * as k8s from "../generic/k8s.js";
import { customHandleSummary } from '../generic/k6_utils.js';

// Parameters
const steveServers = (__ENV.STEVE_SERVERS || 'http://localhost:8080').split(',');
const kubeApiServers = (__ENV.KUBE_SERVERS || 'http://localhost:8080').split(',');
const changeApi = (__ENV.CHANGE_API || 'steve');
const watchApi = (__ENV.WATCH_API || 'steve');
const kubeconfig = k8s.kubeconfig(__ENV.KUBECONFIG, __ENV.CONTEXT)
const namespace = __ENV.NAMESPACE || 'scalability-tests';
const changeRate = parseInt(__ENV.CHANGE_RATE || 1);
const watchMode = __ENV.WATCH_MODE || ''; // "" for full resource, "resource.changes" for notifications
const numConfigMaps = parseInt(__ENV.CONFIG_MAP_COUNT || 100);
const vus = parseInt(__ENV.VUS || 1);
const watchDuration = parseInt(__ENV.WATCH_DURATION || 30);

const username = __ENV.USERNAME;
const password = __ENV.PASSWORD;
const token = __ENV.TOKEN;

const setupTimeout = numConfigMaps / 10;
const setupSettleTime = Math.min(numConfigMaps / 100, 60);
const watchOpenSettleTime = 3;

// Metrics
const deltaFastestSlowest = new Trend('delta_fastest_slowest', true);
const delayFirstObserver = new Trend('delay_first_observer', true);
const delayLastObserver = new Trend('delay_last_observer', true);
const listenerProcessingTime = new Trend('listener_processing_time', true);

export const handleSummary = customHandleSummary;

export const options = {
    insecureSkipTLSVerify: true,
    tlsAuth: [
        {
            cert: kubeconfig["cert"],
            key: kubeconfig["key"],
        },
    ],

    setupTimeout: setupTimeout + "s",

    scenarios: {
        watch: {
            executor: 'per-vu-iterations',
            exec: 'watchScenario',
            vus: 1,
            iterations: 1,
            startTime: setupSettleTime + 's',
            maxDuration: (watchOpenSettleTime + watchDuration) * 1.2 + 's',
        },
        change: {
            executor: 'constant-arrival-rate',
            exec: 'changeScenario',
            rate: changeRate,
            timeUnit: '1s',
            preAllocatedVUs: vus,
            maxVUs: vus,
            startTime: (setupSettleTime + watchOpenSettleTime) + 's',
            duration: watchDuration + 's',
        },
    },

    thresholds: {
        checks: ['rate>0.99'],
        http_req_failed: ['rate<0.01'],
        http_req_duration: ['p(95)<500'],
        delay_first_observer: ['p(95)<500'],
        delay_last_observer: ['p(95)<500'],
        delta_fastest_slowest: ['p(95)<500'],
    },
};

export function setup() {
    console.log('Setting up test');

    if (changeApi !== 'steve' && changeApi !== 'kube') {
        fail("Please specify either 'steve' or 'kube' for CHANGE_API")
    }

    if (watchApi !== 'steve' && watchApi !== 'kube') {
        fail("Please specify either 'steve' or 'kube' for WATCH_API")
    }

    var cookies = {}
    if (token) {
        console.log('Using token for authentication');
        cookies = {R_SESS: token}
    }
    else if (username && password) {
        console.log(`Logging in as ${username}`)
        const res = http.post(`${steveServers[0]}/v3-public/localProviders/local?action=login`, JSON.stringify({
            "description": "UI session",
            "responseType": "cookie",
            "username": username,
            "password": password
        }))

        check(res, {
            'logging in returns status 200': (r) => r.status === 200,
        })

        cookies = http.cookieJar().cookiesForURL(res.url)
    }
    else {
        fail("Please specify either USERNAME and PASSWORD or TOKEN")
    }

    // Clean up any leftovers from past runs
    teardown({ cookies: cookies })

    // Create namespace
    console.log(`Creating namespace ${namespace}`)
    const nsBody = {
        "type": "namespace",
        "metadata": {
            "name": namespace,
        },
    }
    let res = http.post(`${steveServers[0]}/v1/namespaces`, JSON.stringify(nsBody), { cookies: cookies, headers: { "Content-Type": "application/json" } })
    check(res, {
        'create namespace returns 201': (r) => r.status === 201,
    })

    // Create configmaps
    console.log(`Creating ${numConfigMaps} configmaps`)
    for (let i = 0; i < numConfigMaps; i++) {
        const name = `test-config-map-${i}`
        const cmBody = {
            "type": "configmap",
            "metadata": {
                "name": name,
                "namespace": namespace
            },
            "data": {"data": "initial"}
        }
        res = http.post(`${steveServers[0]}/v1/configmaps`, JSON.stringify(cmBody), { cookies: cookies, headers: { "Content-Type": "application/json" } })
        check(res, {
            'create configmap returns 201': (r) => r.status === 201,
        })
    }

    return { cookies: cookies };
}

export function teardown(data) {
    console.log('Tearing down test');
    http.del(`${steveServers[0]}/v1/namespaces/${namespace}`, null, { cookies: data.cookies })
    console.log('Teardown complete');
}

let changeEvents = {};

export async function watchScenario(data) {

    const servers = watchApi === 'steve' ? steveServers : kubeApiServers;

    for (const server of servers) {

        const url = watchApi === 'steve' ?
            `${server.replace('http', 'ws')}/v1/subscribe` :
            `${server.replace('http', 'ws')}/api/v1/namespaces/${namespace}/configmaps?watch=true`;

        console.log(`Connecting to ${url}`);

        let params = {}
        if (watchApi === 'steve') {
            const jar = http.cookieJar();
            jar.set(server, "R_SESS", data.cookies["R_SESS"]);
            params = { jar: jar }
        }
        else {
            // !!! DO NOT REMOVE THIS !!!
            // golang.org/x/net/websocket, the implementation of websocket used in apiserver, checks Origin by default
            // and requires it to be a valid URL (else it returns a 403). Any valid URL will do!
            // https://cs.opensource.google/go/x/net/+/refs/tags/v0.43.0:websocket/server.go;drc=19fe7f4f42382191e644fa98c76c915cd1815487;l=92
            params = { headers: { 'Origin': 'https://just-do.it' },}
        }
        const ws = new WebSocket(url, null, params);

        ws.addEventListener('open', () => {
            console.log(`Connected to ${url}`);

            setTimeout(() => {
                console.log(`Closing socket to ${url}`);
                ws.close();
            }, (watchDuration + setupSettleTime * 2) * 1000);

            if (watchApi === 'steve') {
                ws.send(JSON.stringify({
                    resourceType: 'configmaps',
                    namespace: namespace,
                    mode: watchMode,
                }));
            }
        });

        ws.addEventListener('message', (e) => {
            const now = new Date().getTime();
            const event = JSON.parse(e.data);
            if ((watchApi === 'steve' && event.name === 'resource.change') || (watchApi === 'kube' && event.type === 'MODIFIED')) {
                const data = watchApi === 'steve' ? event.data.data : event.object.data;

                const id = data.id;
                const delay = now - parseInt(data.timestamp);
                if (!changeEvents[id]) {
                    // this is the first server processing the event for this resource
                    changeEvents[id] = [];

                    delayFirstObserver.add(delay)
                }
                changeEvents[id].push(delay);
                console.debug(`Server ${changeEvents[id].length}/${steveServers.length} caught up on ${id}. Delay: ${delay}ms`);

                const events = changeEvents[id];
                if (events.length === steveServers.length) {
                    delayLastObserver.add(delay);

                    const first = events[0];
                    const last = events[events.length - 1];
                    deltaFastestSlowest.add(last - first);
                    console.debug(`Delta between fastest and slowest: ${last - first}ms`);
                    delete changeEvents[id];
                }
            }
            listenerProcessingTime.add(new Date().getTime() - now);
        });

        ws.addEventListener('close', () => console.log(`disconnected from ${url}`));
        ws.addEventListener('error', (e) => {
            if (e.error !== 'websocket: close sent') {
                console.log('An unexpected error occured: ', e.error);
            }
        });
    }
}

export function changeScenario(data) {
    const configMapId = Math.floor(Math.random() * numConfigMaps);
    const name = `test-config-map-${configMapId}`;

    const servers = changeApi === 'steve' ? steveServers : kubeApiServers;
    const server = servers[Math.floor(Math.random() * servers.length)];
    const url = changeApi === 'steve' ?
        `${server}/v1/configmaps/${namespace}/${name}` :
        `${server}/api/v1/namespaces/${namespace}/configmaps/${name}`;
    const cookies = changeApi === 'steve' ? data.cookies : [];

    const getRes = http.get(url, {cookies: cookies});
    if (getRes.status !== 200) {
        fail(`Failed to get configmap ${name}: ${getRes.status} ${getRes.body}`);
    }
    const configmap = JSON.parse(getRes.body);

    configmap.data.id = `${__VU}-${__ITER}`;
    configmap.data.timestamp = `${new Date().getTime()}`;

    const putRes = http.put(url, JSON.stringify(configmap), {
        headers: { 'Content-Type': 'application/json' },
        cookies: cookies
    });
    check(putRes, {
        'update configmap returns 200': (r) => r.status === 200,
    });
    console.debug(`Changed configmap ${name}`);
}

export function handleSummary(data) {
    console.log('Generating summary');
    return {
        'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    };
}
