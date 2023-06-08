import { sleep } from 'k6';
import { Counter } from 'k6/metrics';
import * as k8s from './k8s.js'

// Parameters
const namespace = "scalability-test"

// Option setting
const kubeconfig = k8s.kubeconfig(__ENV.KUBECONFIG, __ENV.CONTEXT)
const baseUrl = __ENV.BASE_URL
const configMapCount = Number(__ENV.CONFIG_MAP_COUNT)
const vus = Number(__ENV.VUS)
const rate = Number(__ENV.RATE)
const duration = __ENV.DURATION

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
        change: {
            executor: 'constant-vus',
            exec: 'change',
            vus: vus,
            duration: duration,
        },
    },
    thresholds: {
        checks: ['rate>0.99']
    }
};

// Custom metrics
const resourceMetric = new Counter('changed_resources')

// Test functions, in order of execution

export function change() {
    const name = `test-config-maps-${Math.floor(Math.random() * configMapCount)}`
    const body = {
        "metadata": {
            "name": name,
            "namespace": namespace
        },
        "data": {"data": (Math.random() + 1).toString(36).substring(2)}
    }

    k8s.update(`${baseUrl}/api/v1/namespaces/${namespace}/configmaps/${name}`, body)
    sleep(vus/rate)

    resourceMetric.add(1)
}
