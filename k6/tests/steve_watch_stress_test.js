import { check, fail, sleep } from 'k6';
import http from 'k6/http';
import encoding from 'k6/encoding';
import exec from 'k6/execution';
import { Trend, Counter } from 'k6/metrics';
import { WebSocket } from 'k6/experimental/websockets';
import { customHandleSummary } from '../generic/k6_utils.js';
import * as k8s from "../generic/k8s.js";

// Steve Watch Stress Test with SQLite Caching
// Stresses Steve's watch functionality with SQLite caching enabled
// Based on: https://gist.github.com/aruiz14/cf279761268a1458cb3838e6f41388ac

const steveUrl = __ENV.STEVE_URL || 'http://localhost:8080';
const kubeApiUrl = __ENV.KUBE_API_URL || k8s.kubeconfig.url;
const namespace = __ENV.NAMESPACE || 'test-configmaps';
const count = parseInt(__ENV.COUNT || 2000);
const watchDuration = parseInt(__ENV.WATCH_DURATION || 600);
const username = __ENV.USERNAME;
const password = __ENV.PASSWORD;
const token = __ENV.TOKEN;
const rancherNamespace = __ENV.RANCHER_NAMESPACE || 'cattle-system';
const rancherPodLabel = __ENV.RANCHER_POD_LABEL || 'app=rancher';

const dataBlob = encoding.b64encode('a'.repeat(750 * 1024));

const steveResponseTime = new Trend('steve_light_read_duration', true);
const walSize = new Trend('sqlite_wal_size_bytes', true);
const watcherErrors = new Counter('watcher_errors');
const eventCreateErrors = new Counter('event_create_errors');
const crdUpdateErrors = new Counter('crd_update_errors');

export const handleSummary = customHandleSummary;

export const options = {
    insecureSkipTLSVerify: true,
    tlsAuth: [
        {
            cert: k8s.kubeconfig.cert,
            key: k8s.kubeconfig.key,
        },
    ],

    setupTimeout: '300s',
    teardownTimeout: '300s',

    scenarios: {
        watchers: {
            executor: 'per-vu-iterations',
            exec: 'watcherScenario',
            vus: count,
            iterations: 1,
            startTime: '10s',
            maxDuration: `${watchDuration + 60}s`,
        },
        createDeleteEvents: {
            executor: 'constant-arrival-rate',
            exec: 'createDeleteEventsScenario',
            rate: 10,
            timeUnit: '1s',
            duration: `${watchDuration}s`,
            preAllocatedVUs: 10,
            maxVUs: 50,
            startTime: '15s',
        },
        updateCRDs: {
            executor: 'constant-arrival-rate',
            exec: 'updateCRDScenario',
            rate: 0.33,
            timeUnit: '1s',
            duration: `${watchDuration}s`,
            preAllocatedVUs: 1,
            maxVUs: 5,
            startTime: '15s',
        },
        lightReadTest: {
            executor: 'constant-arrival-rate',
            exec: 'lightReadScenario',
            rate: 1,
            timeUnit: '1s',
            duration: `${watchDuration}s`,
            preAllocatedVUs: 1,
            maxVUs: 5,
            startTime: '15s',
        },
        checkWALSize: {
            executor: 'constant-arrival-rate',
            exec: 'checkWALSizeScenario',
            rate: 0.1,
            timeUnit: '1s',
            duration: `${watchDuration}s`,
            preAllocatedVUs: 1,
            maxVUs: 2,
            startTime: '15s',
        },
    },

    thresholds: {
        checks: ['rate>0.95'],
        http_req_failed: ['rate<0.05'],
        steve_light_read_duration: ['p(95)<100'],
        sqlite_wal_size_bytes: ['max<10485760'],
    },
};

export function setup() {
    console.log('Setting up Steve watch stress test');

    let cookies = {};
    if (token) {
        console.log('Using token for authentication');
        cookies = {R_SESS: token}
    } else if (username && password) {
        console.log(`Logging in as ${username}`);
        const res = http.post(`${steveUrl}/v3-public/localProviders/local?action=login`, JSON.stringify({
            "description": "UI session",
            "responseType": "cookie",
            "username": username,
            "password": password
        }), {
            headers: { "Content-Type": "application/json" }
        });

        check(res, {
            'logging in returns status 200': (r) => r.status === 200,
        });

        if (res.status !== 200) {
            fail(`Failed to login: ${res.status} ${res.body}`);
        }

        cookies = http.cookieJar().cookiesForURL(res.url);
    } else {
        fail("Please specify either USERNAME and PASSWORD or TOKEN");
    }

    // Clean up any leftovers from past runs
    console.log('Cleaning up previous test resources');
    k8s.del(`${kubeApiUrl}/api/v1/namespaces/${namespace}`);
    sleep(5);

    // Create namespace
    console.log(`Creating namespace ${namespace}`);
    const nsBody = {
        "metadata": {
            "name": namespace,
        },
    };
    const nsRes = k8s.create(`${kubeApiUrl}/api/v1/namespaces`, nsBody);
    check(nsRes, {
        'create namespace succeeds': (r) => r.status === 201 || r.status === 409,
    });

    // Create CRD
    console.log('Creating Custom Resource Definition');
    const crdBody = {
        "apiVersion": "apiextensions.k8s.io/v1",
        "kind": "CustomResourceDefinition",
        "metadata": {
            "name": "foos.example.com"
        },
        "spec": {
            "conversion": {
                "strategy": "None"
            },
            "group": "example.com",
            "names": {
                "kind": "Foo",
                "listKind": "FooList",
                "plural": "foos",
                "singular": "foo"
            },
            "scope": "Cluster",
            "versions": [{
                "additionalPrinterColumns": [{
                    "jsonPath": ".metadata.name",
                    "name": "Name",
                    "type": "string"
                }],
                "name": "v1",
                "schema": {
                    "openAPIV3Schema": {
                        "type": "object"
                    }
                },
                "served": true,
                "storage": true
            }]
        }
    };
    const crdRes = k8s.create(`${kubeApiUrl}/apis/apiextensions.k8s.io/v1/customresourcedefinitions`, crdBody);
    check(crdRes, {
        'create CRD succeeds': (r) => r.status === 201 || r.status === 409,
    });

    sleep(5); // Let CRD settle

    // Create initial test resources (configmap, secret, and custom resource)
    console.log('Creating initial test resources');
    const cmBody = {
        "metadata": {
            "name": "foo",
            "namespace": namespace
        },
        "data": {}
    };
    k8s.create(`${kubeApiUrl}/api/v1/namespaces/${namespace}/configmaps`, cmBody);

    const secretBody = {
        "metadata": {
            "name": "foo",
            "namespace": namespace
        },
        "type": "Opaque",
        "data": {}
    };
    k8s.create(`${kubeApiUrl}/api/v1/namespaces/${namespace}/secrets`, secretBody);

    const fooBody = {
        "apiVersion": "example.com/v1",
        "kind": "Foo",
        "metadata": {
            "name": "foo"
        }
    };
    k8s.create(`${kubeApiUrl}/apis/example.com/v1/foos`, fooBody);

    sleep(5);

    // Get initial resource versions for watchers
    const configmapRV = getResourceVersion(`${steveUrl}/v1/configmaps?filter=metadata.namespace=${namespace}`, cookies);
    const secretRV = getResourceVersion(`${steveUrl}/v1/secrets?filter=metadata.namespace=${namespace}`, cookies);
    
    console.log('Setup complete');
    console.log(`ConfigMap RV: ${configmapRV}, Secret RV: ${secretRV}`);

    return {
        cookies: cookies,
        configmapRV: configmapRV,
        secretRV: secretRV
    };
}

export function teardown(data) {
    console.log('Tearing down test');
    k8s.del(`${kubeApiUrl}/api/v1/namespaces/${namespace}`);
    k8s.del(`${kubeApiUrl}/apis/apiextensions.k8s.io/v1/customresourcedefinitions/foos.example.com`);
    console.log('Teardown complete');
}

function getResourceVersion(url, cookies) {
    const res = http.get(url, { cookies: cookies });
    if (res.status !== 200) {
        console.warn(`Failed to get resource version from ${url}: ${res.status}`);
        return '';
    }
    const data = JSON.parse(res.body);
    return data.revision || '';
}

export function watcherScenario(data) {
    const vuId = exec.vu.idInTest;
    const wsUrl = steveUrl.replace('http', 'ws') + '/v1/subscribe';
    
    try {
        const jar = http.cookieJar();
        jar.set(steveUrl, "R_SESS", data.cookies["R_SESS"]);
        
        const ws = new WebSocket(wsUrl, null, { jar: jar });
        let connected = false;

        ws.addEventListener('open', () => {
            console.debug(`[Watcher ${vuId}] Connected`);
            connected = true;

            ws.send(JSON.stringify({
                resourceType: 'configmaps',
                mode: 'resource.changes',
                debounceMs: 4000,
                resourceVersion: data.configmapRV,
            }));

            ws.send(JSON.stringify({
                resourceType: 'secrets',
                mode: 'resource.changes',
                debounceMs: 4000,
                resourceVersion: data.secretRV,
            }));

            ws.send(JSON.stringify({
                resourceType: 'example.com.foos',
                mode: 'resource.changes',
                debounceMs: 4000,
            }));

            console.debug(`[Watcher ${vuId}] Subscribed`);
        });

        ws.addEventListener('error', (e) => {
            if (e.error !== 'websocket: close sent') {
                console.error(`[Watcher ${vuId}] Error: ${e.error}`);
                watcherErrors.add(1);
            }
        });

        ws.addEventListener('close', () => {
            console.debug(`[Watcher ${vuId}] Disconnected`);
        });

        const jitterPercent = 0.05;
        const jitter = (Math.random() - 0.5) * 2 * watchDuration * jitterPercent;
        sleep(watchDuration + jitter);

        if (connected) {
            ws.close();
        }
    } catch (e) {
        console.error(`[Watcher ${vuId}] Exception: ${e}`);
        watcherErrors.add(1);
    }
}

export function createDeleteEventsScenario() {
    try {
        k8s.create(`${kubeApiUrl}/api/v1/namespaces/${namespace}/configmaps`, {
            "metadata": { "name": "foo", "namespace": namespace },
            "data": { "1m": dataBlob }
        }, false);
        
        k8s.create(`${kubeApiUrl}/api/v1/namespaces/${namespace}/secrets`, {
            "metadata": { "name": "foo", "namespace": namespace },
            "type": "Opaque",
            "data": { "1m": dataBlob }
        }, false);

        k8s.create(`${kubeApiUrl}/apis/example.com/v1/foos`, {
            "apiVersion": "example.com/v1",
            "kind": "Foo",
            "metadata": { "name": "foo" }
        }, false);

        sleep(0.1);

        const delCm = k8s.del(`${kubeApiUrl}/api/v1/namespaces/${namespace}/configmaps/foo`);
        const delSecret = k8s.del(`${kubeApiUrl}/api/v1/namespaces/${namespace}/secrets/foo`);
        const delFoo = k8s.del(`${kubeApiUrl}/apis/example.com/v1/foos/foo`);

        const success = check(null, {
            'create/delete cycle completed': () => true,
            'configmap deleted': () => delCm.status === 200 || delCm.status === 404,
            'secret deleted': () => delSecret.status === 200 || delSecret.status === 404,
            'custom resource deleted': () => delFoo.status === 200 || delFoo.status === 404,
        });

        if (!success) {
            eventCreateErrors.add(1);
        }
    } catch (e) {
        console.error(`Create/Delete error: ${e}`);
        eventCreateErrors.add(1);
    }
}

export function updateCRDScenario() {
    try {
        const useVersion1 = exec.scenario.iterationInTest % 2 === 0;

        const crdBody = {
            "apiVersion": "apiextensions.k8s.io/v1",
            "kind": "CustomResourceDefinition",
            "metadata": {
                "name": "foos.example.com"
            },
            "spec": {
                "conversion": {
                    "strategy": "None"
                },
                "group": "example.com",
                "names": {
                    "kind": "Foo",
                    "listKind": "FooList",
                    "plural": "foos",
                    "singular": "foo"
                },
                "scope": "Cluster",
                "versions": useVersion1 ? [{
                    "additionalPrinterColumns": [{
                        "jsonPath": ".metadata.name",
                        "name": "Name",
                        "type": "string"
                    }],
                    "name": "v1",
                    "schema": {
                        "openAPIV3Schema": {
                            "type": "object"
                        }
                    },
                    "served": true,
                    "storage": true
                }] : [{
                    "name": "v1",
                    "schema": {
                        "openAPIV3Schema": {
                            "type": "object"
                        }
                    },
                    "served": true,
                    "storage": true
                }]
            }
        };

        const url = `${kubeApiUrl}/apis/apiextensions.k8s.io/v1/customresourcedefinitions/foos.example.com`;
        const res = http.put(url, JSON.stringify(crdBody), {
            headers: { 'Content-Type': 'application/json' }
        });

        const success = check(res, {
            'CRD update succeeds': (r) => r.status === 200,
        });

        if (!success) {
            console.error(`CRD update failed: ${res.status} ${res.body}`);
            crdUpdateErrors.add(1);
        }

        sleep(3);
    } catch (e) {
        console.error(`CRD update error: ${e}`);
        crdUpdateErrors.add(1);
    }
}

export function lightReadScenario(data) {
    const startTime = new Date().getTime();
    const url = `${steveUrl}/v1/configmaps?filter=metadata.namespace=${namespace}`;
    const res = http.get(url, { cookies: data.cookies });
    const duration = new Date().getTime() - startTime;
    steveResponseTime.add(duration);
    check(res, {
        'light read returns 200': (r) => r.status === 200,
    });
}

export function checkWALSizeScenario() {
    try {
        const podsUrl = `${kubeApiUrl}/api/v1/namespaces/${rancherNamespace}/pods?labelSelector=${encodeURIComponent(rancherPodLabel)}`;
        const podsRes = http.get(podsUrl);
        
        if (podsRes.status !== 200) {
            console.warn(`Failed to get Rancher pods: ${podsRes.status}`);
            return;
        }

        const pods = JSON.parse(podsRes.body);
        if (!pods.items || pods.items.length === 0) {
            console.warn('No Rancher pods found');
            return;
        }

        const podName = pods.items[0].metadata.name;
        const walPath = '/var/lib/rancher/informer_object_cache.db-wal';
        const cmd = `if [ -f ${walPath} ]; then stat -c %s ${walPath} 2>/dev/null || stat -f %z ${walPath} 2>/dev/null; else echo 0; fi`;
        
        const result = k8s.exec(kubeApiUrl, rancherNamespace, podName, 'rancher', cmd, 10);
        
        if (result.success && result.stdout) {
            const size = parseInt(result.stdout);
            if (!isNaN(size)) {
                walSize.add(size);
                if (size > 10485760) {
                    console.warn(`WAL size exceeds 10MB: ${size} bytes`);
                }
            }
        }
    } catch (e) {
        console.error(`WAL size check error: ${e}`);
    }
}
