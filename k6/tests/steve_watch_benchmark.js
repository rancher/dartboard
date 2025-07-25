
import { check, fail, sleep } from 'k6';
import http from 'k6/http';
import { Trend } from 'k6/metrics';
import ws from 'k6/ws';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';

// Parameters
const steveServers = (__ENV.STEVE_SERVERS || 'http://localhost:8080').split(',');
const namespace = __ENV.NAMESPACE || 'scalability-tests';
const resource = __ENV.RESOURCE || 'configmaps';
const changeRate = __ENV.CHANGE_RATE || 1;
const watchMode = __ENV.WATCH_MODE || ''; // "" for full resource, "resource.changes" for notifications
const numConfigMaps = __ENV.CONFIG_MAP_COUNT || 100;
const vus = __ENV.K6_VUS || 1;
const watchDuration = __ENV.WATCH_DURATION || 30;

const username = __ENV.USERNAME;
const password = __ENV.PASSWORD;
const token = __ENV.TOKEN;


// Metrics
const deltaFastestSlowest = new Trend('delta_fastest_slowest', true);
const delayFirstObserver = new Trend('delay_first_observer', true);
const delayLastObserver = new Trend('delay_last_observer', true);

export const options = {
    insecureSkipTLSVerify: true,
    scenarios: {
        watch: {
            executor: 'per-vu-iterations',
            exec: 'watchScenario',
            vus: vus,
            iterations: 1,
            maxDuration: watchDuration + 's',
        },
        change: {
            executor: 'constant-arrival-rate',
            exec: 'changeScenario',
            rate: changeRate,
            timeUnit: '1s',
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
    var cookies = {}
    if (token) {
        cookies = {R_SESS: token}
    }
    else if (username && password) {
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
    http.del(`${steveServers[0]}/v1/namespaces/${namespace}`, null, { cookies: data.cookies })
}

let lastChangeTime = null;
let changeEvents = {};

export function watchScenario(data) {
    const sockets = [];
    steveServers.forEach(server => {
        const url = server.replace('http', 'ws') + '/v1/subscribe';
        const jar = http.cookieJar();
        jar.set(server, "R_SESS", data.cookies["R_SESS"]);
        const res = ws.connect(url, {jar: jar}, function(socket) {
            socket.on('open', () => {
                socket.send(JSON.stringify({
                    resourceType: resource,
                    namespace: namespace,
                    mode: watchMode,
                }));
            });

            socket.on('message', (message) => {
                const event = JSON.parse(message);
                if (event.name === 'resource.change') {
                    const now = new Date().getTime();
                    const resourceName = event.data.metadata.name;
                    if (!changeEvents[resourceName]) {
                        changeEvents[resourceName] = [];
                    }
                    changeEvents[resourceName].push(now);

                    if (changeEvents[resourceName].length === steveServers.length) {
                        const events = changeEvents[resourceName];
                        const first = Math.min(...events);
                        const last = Math.max(...events);
                        deltaFastestSlowest.add(last - first);
                        if(lastChangeTime) {
                            delayFirstObserver.add(first - lastChangeTime);
                            delayLastObserver.add(last - lastChangeTime);
                        }
                        delete changeEvents[resourceName];
                    }
                }
            });

            socket.on('close', () => console.log(`disconnected from ${url}`));
            socket.on('error', function (e) {
                if (e.error() != 'websocket: close sent') {
                    console.log('An unexpected error occured: ', e.error());
                }
            });

            socket.setTimeout(function () {
                console.log(`Closing socket to ${url}`)
                socket.close();
            }, watchDuration * 1000);
        });
        check(res, { 'status is 101': (r) => r && r.status === 101 });
        sockets.push(res);
    });

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

    configmap.data.data = `updated-${new Date().getTime()}`;

    lastChangeTime = new Date().getTime();
    const putRes = http.put(`${server}/v1/configmaps/${namespace}/${name}`, JSON.stringify(configmap), {
        headers: { 'Content-Type': 'application/json' },
        cookies: data.cookies
    });
    check(putRes, {
        'update configmap returns 200': (r) => r.status === 200,
    });
}

export function handleSummary(data) {
    return {
        'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    };
}
