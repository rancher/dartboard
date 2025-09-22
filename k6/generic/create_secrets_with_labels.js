import encoding from 'k6/encoding';
import exec from 'k6/execution';
import * as k8s from './k8s.js'

// Parameters
const namespace = "scalability-test"
const count = Number(__ENV.COUNT)
const data = encoding.b64encode("a".repeat(10*240))
const vus = 1

// Option setting
const kubeconfig = k8s.kubeconfig(__ENV.KUBECONFIG, __ENV.CONTEXT)
const baseUrl = kubeconfig["url"]

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
        createKeyOne: {
            executor: 'shared-iterations',
            exec: 'createSecretsWithLabels',
            vus: vus,
            iterations: count,
            maxDuration: '1h',
        },
    },
    thresholds: {
        checks: ['rate>0.99']
    }
}

// Test functions, in order of execution
export function createSecretsWithLabels() {
    const name = `test-secret-${exec.scenario.iterationInTest}e`
    const key_1 = "cow"
    const value = "geeko"
    const body = {
        "metadata": {
            "name": name,
            "namespace": namespace,
            "labels": {[key_1]:value}
        },
        "data": {"data": data},
    }

    k8s.create(`${baseUrl}/api/v1/namespaces/${namespace}/secrets`, body)
}
