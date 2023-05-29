import { sleep } from 'k6';
import encoding from 'k6/encoding';
import exec from 'k6/execution';
import * as k8s from './k8s.js'

// Parameters
const namespace = "scalability-test-temp"
const data = encoding.b64encode("a".repeat(1))
const vus = 5
const duration = '2h'
const rate = 1

// Option setting
const kubeconfig = k8s.kubeconfig(__ENV.KUBECONFIG, __ENV.CONTEXT)
const baseUrl = __ENV.BASE_URL

export const options = {
    insecureSkipTLSVerify: true,
    tlsAuth: [
        {
            cert: kubeconfig["cert"],
            key: kubeconfig["key"],
        },
    ],

    summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(95)', 'p(99)', 'count'],

    scenarios: {
        create: {
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

    k8s.create(`${baseUrl}/api/v1/namespaces/${namespace}/configmaps`, body)
    sleep(1.0/rate)
    k8s.del(`${baseUrl}/api/v1/namespaces/${namespace}/configmaps/${name}`)
}
