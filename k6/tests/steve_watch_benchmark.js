import { check, fail, sleep } from 'k6';
import http from 'k6/http';
import { Trend } from 'k6/metrics';
import { WebSocket } from 'k6/experimental/websockets';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';

// Parameters
const steveServers = (__ENV.STEVE_SERVERS || 'http://localhost:8080').split(',');
const namespace = __ENV.NAMESPACE || 'scalability-tests';
const resource = __ENV.RESOURCE || 'configmaps';
const changeRate = parseInt(__ENV.CHANGE_RATE || 1);
const watchMode = __ENV.WATCH_MODE || ''; // "" for full resource, "resource.changes" for notifications
const numConfigMaps = parseInt(__ENV.CONFIG_MAP_COUNT || 100);
const vus = parseInt(__ENV.VUS || 1);
const watchDuration = parseInt(__ENV.WATCH_DURATION || 30);

const username = __ENV.USERNAME;
const password = __ENV.PASSWORD;
const token = __ENV.TOKEN;

const setupTimeout = numConfigMaps / 10;
const setupSettleTime = numConfigMaps * 0.01;
const watchOpenSettleTime = 3;

// Metrics
const deltaFastestSlowest = new Trend('delta_fastest_slowest', true);
const delayFirstObserver = new Trend('delay_first_observer', true);
const delayLastObserver = new Trend('delay_last_observer', true);

export const options = {
    insecureSkipTLSVerify: true,
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
            startTime: (setupSettleTime + watchOpenSettleTime) + 's',
            duration: watchDuration + 's',
            preAllocatedVUs: 10,
        },
    },

    thresholds: {
        checks: ['rate>0.99'],
        http_req_failed: ['rate<0.01'],
        http_req_duration: ['p(95)<500'],
    },
};

export function setup() {
    console.log('Setting up test');
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
    console.log('Starting watch scenario');

    for (const server of steveServers) {
        const url = server.replace('http', 'ws') + '/v1/subscribe';
        console.log(`Connecting to ${url}`);
        const jar = http.cookieJar();
        jar.set(server, "R_SESS", data.cookies["R_SESS"]);
        const ws = new WebSocket(url, { jar: jar });

        ws.addEventListener('open', () => {
            console.log(`Connected to ${url}`);

            setTimeout(() => {
                console.log(`Closing socket to ${url}`);
                ws.close();
            }, (watchDuration + setupSettleTime * 2) * 1000);

            ws.send(JSON.stringify({
                resourceType: resource,
                namespace: namespace,
                mode: watchMode,
            }));
        });

        ws.addEventListener('message', (e) => {
            const event = JSON.parse(e.data);
            if (event.name === 'resource.change') {
                const now = new Date().getTime();
                const delay = now - parseInt(event.data.data.data);
                const resourceName = event.data.metadata.name;
                if (!changeEvents[resourceName]) {
                    // this is the first server processing the event for this resource
                    changeEvents[resourceName] = [];

                    delayFirstObserver.add(delay)
                }
                changeEvents[resourceName].push(now);
                console.log(`Server ${changeEvents[resourceName].length}/${steveServers.length} caught up on ${resourceName}. Delay: ${delay}ms`);

                if (changeEvents[resourceName].length === steveServers.length) {
                    delayLastObserver.add(delay);

                    const events = changeEvents[resourceName];
                    const first = Math.min(...events);
                    const last = Math.max(...events);
                    deltaFastestSlowest.add(last - first);
                    console.log(`Delta between fastest and slowest: ${last - first}ms`);
                    delete changeEvents[resourceName];
                }
            }
        });

        ws.addEventListener('close', () => console.log(`disconnected from ${url}`));
        ws.addEventListener('error', (e) => {
            if (e.error != 'websocket: close sent') {
                console.log('An unexpected error occured: ', e.error);
            }
        });
    }

    console.log("Done watching");
}

export function changeScenario(data) {
    const configMapId = Math.floor(Math.random() * numConfigMaps);
    const serverId = Math.floor(Math.random() * steveServers.length);
    const server = steveServers[serverId];
    const name = `test-config-map-${configMapId}`;

    const getRes = http.get(`${server}/v1/configmaps/${namespace}/${name}`, {cookies: data.cookies});
    if (getRes.status !== 200) {
        fail(`Failed to get configmap ${name}: ${getRes.status} ${getRes.body}`);
    }
    const configmap = JSON.parse(getRes.body);

    configmap.data.data = `${new Date().getTime()}`;

    const putRes = http.put(`${server}/v1/configmaps/${namespace}/${name}`, JSON.stringify(configmap), {
        headers: { 'Content-Type': 'application/json' },
        cookies: data.cookies
    });
    check(putRes, {
        'update configmap returns 200': (r) => r.status === 200,
    });
    console.log(`Changed configmap ${name}`);
}

export function handleSummary(data) {
    console.log('Generating summary');
    return {
        'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    };
}
