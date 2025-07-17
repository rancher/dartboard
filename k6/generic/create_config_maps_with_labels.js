import encoding from 'k6/encoding';
import exec from 'k6/execution';
import * as k8s from './k8s.js'

// Parameters
const namespace = "scalability-test"
const count = Number(__ENV.COUNT)
const data = encoding.b64encode("a".repeat(10*1024))
const vus = 10

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
        create: {
            executor: 'shared-iterations',
            exec: 'create',
            vus: vus,
            iterations: count,
            maxDuration: '1h',
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

export function create() {
    const name = `test-config-map-${exec.scenario.name}-${exec.scenario.iterationInTest}`

    let key_1 = "cow"
    let value = "geeko"
    if (`${exec.scenario.iterationInTest}` >= (count/2)) {
        key_1 = "blue"
        value = "green"
    }

    const body = {
        "metadata": {
            "name": name,
            "namespace": namespace,
            "labels": {[key_1]:value}
        },
        "data": {"data": data},
    }

    k8s.create(`${baseUrl}/api/v1/namespaces/${namespace}/configmaps`, body)
}
