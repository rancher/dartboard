import { sleep } from 'k6';
import encoding from 'k6/encoding';
import exec from 'k6/execution';
import * as k8s from './k8s.js';
import { customHandleSummary } from './k6_utils.js';

// Parameters
const namespace = "scalability-test-temp"
const data = encoding.b64encode("a".repeat(1))
const duration = '2h'
const vus = __ENV.VUS || 5
// 2 requests per iteration, so iteration rate is half of request rate
const rate =  (__ENV.RATE || 2) / 2

// Option setting
const baseUrl = __ENV.BASE_URL

export const handleSummary = customHandleSummary;

export const options = {
    insecureSkipTLSVerify: true,
    tlsAuth: [
        {
            cert: k8s.kubeconfig["cert"],
            key: k8s.kubeconfig["key"],
        },
    ],

    summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(95)', 'p(99)', 'count'],

    scenarios: {
        change: {
            executor: 'constant-arrival-rate',
            exec: 'change',
            preAllocatedVUs: vus,
            duration: duration,
            rate: rate,
        },
    },
    thresholds: {
        checks: ['rate>0.99']
    }
};

// Test functions, in order of execution

export function setup() {
    // delete leftovers, if any
    k8s.del(`${baseUrl}/api/v1/namespaces/${namespace}`)

    // create empty namespace
    const body = {
        "metadata": {
            "name": namespace,
        },
    }
    k8s.create(`${baseUrl}/api/v1/namespaces`, body)
}

export function change() {
    const name = `test-config-map-${exec.scenario.name}-${exec.scenario.iterationInTest}`
    const body = {
        "metadata": {
            "name": name,
            "namespace": namespace
        },
        "data": {"data": data}
    }

    k8s.create(`${baseUrl}/api/v1/namespaces/${namespace}/configmaps`, body, false)
    k8s.del(`${baseUrl}/api/v1/namespaces/${namespace}/configmaps/${name}`)
}
