import encoding from 'k6/encoding';
import exec from 'k6/execution';
import { Gauge } from 'k6/metrics';
import * as k8s from './k8s.js'

// Parameters
const namespace = "scalability-test"
const configMapCount = Number(__ENV.CONFIG_MAP_COUNT)
const secretCount = Number(__ENV.SECRET_COUNT)
const data = encoding.b64encode("a".repeat(10*1024))
const vus = 1

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

    setupTimeout: '8h',

    scenarios: {
        createConfigMaps: {
            executor: 'shared-iterations',
            exec: 'createConfigMaps',
            vus: vus,
            iterations: configMapCount,
            maxDuration: '1h',
        },
        createSecrets: {
            executor: 'shared-iterations',
            exec: 'createSecrets',
            vus: vus,
            iterations: secretCount,
            maxDuration: '1h',
        },
    },
    thresholds: {
        checks: ['rate>0.99']
    }
}

// Custom metrics
const resourceMetric = new Gauge('test_resources')

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

export function createConfigMaps() {
    const name = `test-config-maps-${exec.scenario.iterationInTest}`
    const body = {
        "metadata": {
            "name": name,
            "namespace": namespace
        },
        "data": {"data": data}
    }

    k8s.create(`${baseUrl}/api/v1/namespaces/${namespace}/configmaps`, body)
    resourceMetric.add(configMapCount + secretCount)
}

export function createSecrets() {
    const name = `test-secrets-${exec.scenario.iterationInTest}`
    const body = {
        "metadata": {
            "name": name,
            "namespace": namespace
        },
        "data": {"data": data}
    }

    k8s.create(`${baseUrl}/api/v1/namespaces/${namespace}/secrets`, body)
    resourceMetric.add(configMapCount + secretCount)
}
