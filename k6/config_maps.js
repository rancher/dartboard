import { sleep } from 'k6';
import encoding from 'k6/encoding';
import { Trend } from 'k6/metrics';
import { textSummary } from './lib/k6-summary-0.0.2.js';
import papaparse from './lib/papaparse-5.3.2.js';
import * as k8s from './k8s.js'


// Parameters
const namespace = "scalability-test"
const min = 1000
const max = 256000
const data = encoding.b64encode("a".repeat(4))
const vus = 1
const listIterations = 50

// Option setting
const kubeconfig = k8s.kubeconfig(__ENV.KUBECONFIG)
const baseUrl = kubeconfig["url"]
const resourceCounts = exponentialCounts(min, max)
const scenarios = createAndListScenarios(resourceCounts, vus, listIterations)
const thresholds = sanityCheckThresholds(scenarios)
export const options = {
    insecureSkipTLSVerify: true,
    tlsAuth: [
        {
            cert: kubeconfig["cert"],
            key: kubeconfig["key"],
        },
    ],

    summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(95)', 'p(99)', 'count'],

    scenarios: scenarios,
    thresholds: thresholds
};

const k8sReqDurationTrend = new Trend('k8s_req_duration');

// Compute counts starting from min and multiplying times 2 up to max. E.g.
// exponentialCounts(100, 900) === [100, 200, 400, 800, 1600]
function exponentialCounts(min, max) {
    const iterations = 1 + Math.ceil(Math.log2(max/min))
    return Array.from(
        new Array(iterations),
        (x, i) => i === 0 ? min : min * (2 << (i - 1))
    )
}

// Compute scenarios to create and list resources in quantities equal to resourceCounts
function createAndListScenarios(resourceCounts, vus, listIterations) {
    const result = {}
    let cumulatedIterations = 0

    resourceCounts.forEach((count, i) => {
        // resource creation scenario
        const iterations = i === 0 ? count : (count - resourceCounts[i - 1])
        result[`create-${count}`] = {
            executor: 'shared-iterations',
            exec: 'create',
            vus: vus,
            iterations: iterations,
            env: {
                startAfterIteration: cumulatedIterations.toString()
            },
            maxDuration: '24h',
            startTime: cumulatedIterations * 0.001, // HACK: only to show in time order during execution
        }
        cumulatedIterations += iterations

        // resource list scenario
        result[`list-${count}`] = {
            executor: 'shared-iterations',
            exec: 'list',
            vus: vus,
            iterations: listIterations,
            env: {
                startAfterIteration: cumulatedIterations.toString()
            },
            maxDuration: '24h',
            startTime: cumulatedIterations * 0.001, // HACK: only to show in time order during execution
        }
        cumulatedIterations += listIterations
    })

    return result
}

// Returns some minimum sanity check thresholds
function sanityCheckThresholds(scenarios) {
    let result = {
        checks: ['rate>0.99']
    }

    // HACK: force k6 to compute per-scenario summary metrics via an always-true threshold
    // can be removed when https://github.com/grafana/k6/issues/1321 is implemented
    // this workaround follows recommendation in https://github.com/grafana/k6/issues/1836#issuecomment-771576459
    Object.keys(scenarios).forEach(scenarioName => {
        result[`k8s_req_duration{scenario:${scenarioName}}`] = [`max>=0`]
    })

    return result
}

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

import exec from 'k6/execution';

// HACK: force k6 to wait for previous scenarios to finish
// Note: this only really works on single instances, for scenarios with a known number of iterations
// Proper solution can be tackled when this issue is closed: https://github.com/grafana/k6/issues/1342
function waitPreviousScenarios() {
    const startAfterIteration =  parseInt(scenarios[exec.scenario.name].env.startAfterIteration)
    while (true) {
        const currentIteration = exec.instance.iterationsInterrupted + exec.instance.iterationsCompleted
        if (currentIteration >= startAfterIteration) {
            return
        }
        sleep(1)
    }
}

export function create() {
    waitPreviousScenarios()

    const name = `test-config-map-${exec.scenario.name}-${exec.scenario.iterationInTest}`
    const body = {
        "metadata": {
            "name": name,
            "namespace": namespace
        },
        "data": {"data": data}
    }

    const res = k8s.create(`${baseUrl}/api/v1/namespaces/${namespace}/configmaps`, body)
    k8sReqDurationTrend.add(res.timings.duration, {scenario: exec.scenario.name})
}

export function list() {
    waitPreviousScenarios()
    const responses = k8s.list(`${baseUrl}/api/v1/namespaces/${namespace}/configmaps`)
    const totalDuration = responses.reduce((previous,current) => previous + current.timings.duration, 0)
    k8sReqDurationTrend.add(totalDuration, {scenario: exec.scenario.name})
}

export function teardown(data) {
    k8s.del(`${baseUrl}/api/v1/namespaces/${namespace}`)
}

export function handleSummary(data) {
    const headers = ["resources"].concat(options.summaryTrendStats)
    let csv = []

    Array("create", "list").forEach(operation => {
        csv.push([operation])
        csv.push(headers)
        resourceCounts.forEach(count => {
            const stats = data.metrics[`k8s_req_duration{scenario:${operation}-${count}}`].values
            const row = [count].concat(options.summaryTrendStats.map(h => stats[h].toFixed(3)))
            csv.push(row)
        })
    })

    return {
        'stdout': textSummary(data, { indent: ' ', enableColors: true }), // keep default text output
        'results.csv': papaparse.unparse(csv, { delimiter: '\t' }),
    }
}
