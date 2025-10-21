import { sleep, check, fail } from 'k6';
import encoding from 'k6/encoding';
import exec from 'k6/execution';
import * as k8s from '../generic/k8s.js'
import * as diagnosticsUtil from "../rancher/rancher_diagnostics.js";
import { getCookies, login, retryUntilOneOf } from "../rancher/rancher_utils.js";
import { Gauge, Trend, Counter } from 'k6/metrics';
import * as userUtil from "../rancher/rancher_users_utils.js"
import * as namespacesUtil from "../namespaces/namespace_utils.js"
import * as projectsUtil from "../projects/project_utils.js"
import { list as benchmarkList } from "../tests/api_benchmark.js"
import { customHandleSummary } from '../generic/k6_utils.js';

// Parameters
const namespace = "dartboard-test"
const configMapData = encoding.b64encode("a".repeat(1))
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD
const token = __ENV.TOKEN
const resource = __ENV.RESOURCE || "configmaps"

/*
 Due to the import from `api_benchmark.js` various env vars contained in that script
 are also relevant here. Be sure to set those accordingly in your shell session before
 running this script.
*/

// Requires the same parameters as `api_benchmark.js`, but since we now import
// from that script, those env vars get loaded as well.

// Option setting
const listVUs = Number(__ENV.VUS || 5)
// https://grafana.com/docs/k6/latest/using-k6/scenarios/concepts/arrival-rate-vu-allocation/#pre-allocation-in-arrival-rate-executors
const changeVUs = Number(__ENV.PRE_ALLOCATED_VUS || 5)
const duration = (__ENV.DURATION || '2h')
const diagnosticsInterval = (__ENV.DIAGNOSTICS_INTERVAL || "20m"); // # time unit
// 2 requests per iteration (for change() func), so iteration rate is 1/2 of request rate
const changeIPS = (__ENV.TARGET_RPS || 10) / 2
const kubeconfig = k8s.kubeconfig(__ENV.KUBECONFIG, __ENV.CONTEXT)

// Metrics
const diagnosticsBeforeGauge = new Gauge('diagnostics_before_churn');
const diagnosticsDuringTrend = new Trend('diagnostics_during_churn');
const diagnosticsAfterGauge = new Gauge('diagnostics_after_churn');
const churnOpsCounter = new Counter('churn_operations_total');

const useFilters = (__ENV.USE_FILTERS !== undefined && __ENV.USE_FILTERS !== null) ? __ENV.USE_FILTERS === "true" ? true : false : true
let defaultFilters = [
  "sort=metadata.name&",
  "filter=metadata.namespace!=p-4vgxn&",
  "filter=metadata.namespace!=p-5xk4x&",
  "filter=metadata.namespace!=cattle-fleet-clusters-system&",
  "filter=metadata.namespace!=cattle-fleet-local-system&",
  "filter=metadata.namespace!=cattle-fleet-system&",
  "filter=metadata.namespace!=cattle-global-data&",
  "filter=metadata.namespace!=cattle-impersonation-system&",
  "filter=metadata.namespace!=cattle-provisioning-capi-system&",
  "filter=metadata.namespace!=cattle-system&",
  "filter=metadata.namespace!=cattle-ui-plugin-system&",
  "filter=metadata.namespace!=fleet-default&",
  "filter=metadata.namespace!=fleet-local&",
  "filter=metadata.namespace!=kube-node-lease&",
  "filter=metadata.namespace!=kube-public&",
  "filter=metadata.namespace!=kube-system&"
]

export const handleSummary = customHandleSummary;

export const options = {
  insecureSkipTLSVerify: true,
  tlsAuth: [
    {
      cert: kubeconfig["cert"],
      key: kubeconfig["key"],
    },
  ],

  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(95)', 'p(99)', 'count'],
  setupTimeout: "3m",

  scenarios: {
    preChurnDiagnostics: {
      executor: 'shared-iterations',
      exec: 'preChurnDiagnostics',
      vus: 1,
      iterations: 1,
      startTime: '0s',
      maxDuration: '2m',
      tags: { phase: 'pre-churn' }
    },
    change: {
      executor: 'constant-arrival-rate',
      exec: 'change',
      preAllocatedVUs: changeVUs,
      duration: duration,
      rate: changeIPS,
      startTime: '2m',
      tags: { phase: 'churn' },
    },
    list: {
      executor: 'per-vu-iterations',
      exec: 'list',
      vus: listVUs,
      iterations: 9999, // setting this to a # we should never reach in order to ensure `list` runs until `change` scenario completes
      maxDuration: duration,
      startTime: '2m',
      tags: { phase: 'list' },
    },
    duringChurnDiagnostics: {
      executor: 'constant-arrival-rate',
      exec: 'duringChurnDiagnostics',
      preAllocatedVUs: 1,
      rate: 1,
      timeUnit: diagnosticsInterval,
      duration: duration,
      startTime: '2m',
      tags: { phase: 'during-churn' },
    },
    postChurnDiagnostics: {
      executor: 'shared-iterations',
      exec: 'postChurnDiagnostics',
      vus: 1,
      iterations: 1,
      startTime: `${parseDurationToMinutes(duration) + 5}m`,
      maxDuration: '5m',
      tags: { phase: 'post-churn' }
    },
  },
  thresholds: {
    http_req_failed: ['rate<=0.02'], // Across all scenarios, <2% failures
    'http_req_failed{scenario:change}': ['rate<=0.05'], // Allow 5% error rate during churn
    'http_req_failed{scenario:list}': ['rate<=0.05'], // Allow 5% error rate during churn
    'http_req_duration{scenario:change}': ['p(95)<=2000', 'avg<=1000'], // 95% of requests should be below 2s
    'http_req_duration{scenario:list}': ['p(95)<=1500'], // 95% of requests should be below 2s
    checks: ['rate>0.98'], // Overall correctness across test
    'checks{scenario:change}': ['rate>0.95'], // 95% success rate
    'checks{scenario:list}': ['rate>0.95'], // 95% success rate
    'diagnostics_during_churn{scenario:duringChurnDiagnostics}': ['p(95)<=5000'], // 95% of diagnostic runs finish within 5s
    churn_operations_total: [`count>=${changeIPS * parseInt(duration) * 0.9}`], // At least 90% of target ops
  }
};

function parseDurationToMinutes(str) {
  // Define conversion factors to minutes
  const unitToMinutes = {
    s: 1 / 60,    // seconds
    m: 1,         // minutes
    h: 60,        // hours
    d: 1440       // days
  };
  let total = 0;

  // Regex matches number + unit, with 'ms' first to avoid mis-splitting "ms" as "m" + "s"
  const re = /(\d+)(ms|[smhd])/gi;
  let match;

  while ((match = re.exec(str)) !== null) {
    const value = parseInt(match[1], 10);
    const unit = match[2].toLowerCase();

    if (!(unit in unitToMinutes)) {
      throw new Error(`Unknown duration unit “${unit}” in "${str}"`);
    }

    total += value * unitToMinutes[unit];
  }

  // Return the integer number of minutes (floor)
  return Math.floor(total);
}

function cleanup(cookies, namePrefix) {
  let deleteAllFailed = false

  console.log(`Cleaning up created namespaces and projects with prefix: '${namePrefix}'`)
  let { __, namespaceArray } = namespacesUtil.getNamespacesMatchingName(baseUrl, cookies, namePrefix)
  console.log(`Found ${namespaceArray.length} namespaces to clean up`)
  namespaceArray.forEach(r => {
    let delRes = retryUntilOneOf([200, 204], 5, () => namespacesUtil.deleteNamespace(baseUrl, cookies, r["id"]))
    if (delRes.status !== 200 && delRes.status !== 204) deleteAllFailed = true
    sleep(0.1)
  })

  sleep(2)

  let { _, projectArray } = projectsUtil.getNormanProjectsMatchingName(baseUrl, cookies, namePrefix)
  console.log(`Found ${projectArray.length} projects to clean up`)
  projectArray.forEach(r => {
    let delRes = retryUntilOneOf([200, 204], 5, () => projectsUtil.deleteNormanProject(baseUrl, cookies, r["id"]))
    if (delRes.status !== 200 && delRes.status !== 204) deleteAllFailed = true
    sleep(0.1)
  })
  return deleteAllFailed
}

function getProjectWithRetry(baseUrl, cookies, projectId, maxRetries = 5) {
  for (let retry = 0; retry < maxRetries; retry++) {
    let { res, project } = projectsUtil.getProject(baseUrl, cookies, projectId)
    if (res.status == 200 && project && project.status.conditions) {
      return project
    }
    console.warn(`GET Project attempt #${retry + 1} failed with status ${res.status}`)
    sleep(1)
  }
  return {}
}

export function setup() {
  var cookies = {}
  // log in
  // if session cookie was specified, save it
  if (token) {
    cookies = { R_SESS: token }
  } else if (username != "" && password != "") {

    let adminLoginRes = login(baseUrl, {}, username, password)

    if (adminLoginRes.status !== 200) {
      fail(`could not login to cluster as admin`)
    }
    cookies = getCookies(baseUrl)
  } else {
    fail("Must provide token or login credentials")
  }

  // delete leftovers, if any
  cleanup(cookies, namespace)

  const projectBody = JSON.stringify({
    type: "project",
    name: namespace,
    description: `Dartboard test project`,
    clusterId: "local",
    creatorId: userUtil.getCurrentUserPrincipalId(baseUrl, cookies),
    labels: {},
    annotations: {},
    resourceQuota: {},
    containerDefaultResourceLimit: {},
    namespaceDefaultResourceQuota: {}
  })
  let projRes = projectsUtil.createNormanProject(baseUrl, projectBody, cookies)
  if (projRes.status !== 201) {
    fail("Dartboard test project not created")
  }

  const projectData = JSON.parse(projRes.body)
  let projectId = projectData.id.replace(":", "/")

  sleep(15)

  let steveProject = getProjectWithRetry(baseUrl, cookies, projectId, 10)
  if (!steveProject || !steveProject.status.conditions) {
    fail("Dartboard test project could not be retrieved")
  }

  // create empty namespace
  const namespaceBody = JSON.stringify({
    "type": "namespace",
    "disableOpenApiValidation": false,
    "metadata": {
      "name": namespace,
      "labels": {
        "field.cattle.io/projectId": steveProject.metadata.name
      },
      "annotations": {
        "field.cattle.io/containerDefaultResourceLimit": "{}",
        "field.cattle.io/description": `Dartboard test namespace`,
        "field.cattle.io/projectId": projectData.id
      },
    },
  })
  let res = namespacesUtil.createNamespace(baseUrl, namespaceBody, cookies)
  if (res.status !== 201) {
    fail("Dartboard test namespace not created")
  }

  // Build full filter list (defaultFilters + all Project IDs, namespace IDs, etc.)
  let filters = [...defaultFilters] // Copy default filters

  console.log(`Total filters available: ${filters.length}. Filters: ${filters}`)

  return { cookies: cookies, filters: filters }
}

export function preChurnDiagnostics(data) {
  console.log('=== Collecting PRE-CHURN diagnostics ===');

  const resourceCounts = collectResourceCounts(data.cookies, null);
  collectAPITimings(data.cookies, null);

  // Record baseline metrics
  diagnosticsUtil.metrics.forEach(({ key, gauge }) => {
    const count = resourceCounts[key]?.totalCount;
    if (count != null) {
      diagnosticsBeforeGauge.add(count, { resource: key });
    }
  });

  console.log('Pre-churn diagnostics collected');
}

export function change() {
  const name = `test-config-map-${exec.scenario.name}-${exec.scenario.iterationInTest}`
  const body = {
    "metadata": {
      "name": name,
      "namespace": namespace
    },
    "data": { "data": configMapData }
  }

  k8s.create(`${kubeconfig.url}/api/v1/namespaces/${namespace}/configmaps`, body, false)
  churnOpsCounter.add(1, { resource: resource });
  k8s.del(`${kubeconfig.url}/api/v1/namespaces/${namespace}/configmaps/${name}`)
  churnOpsCounter.add(1, { resource: resource });
}

export function list(data) {
  // need to remove the final filter's '&' suffix so that the list url is formed correctly
  data.filters[data.filters.length - 1] = data.filters[data.filters.length - 1].replace("&", "")
  let allFilters = useFilters ? data.filters.join("") : ""
  benchmarkList(data.cookies, allFilters)
}

export function duringChurnDiagnostics(data) {
  console.log('=== Collecting DURING-CHURN diagnostics ===');

  const start = Date.now();
  collectResourceCounts(data.cookies, null);
  collectAPITimings(data.cookies, null);
  const duration = Date.now() - start;

  // Record timing for diagnostic collection overhead
  diagnosticsDuringTrend.add(duration);

  console.log(`During-churn diagnostics collected in ${duration}ms`);
}

export function postChurnDiagnostics(data) {
  console.log('=== Collecting POST-CHURN diagnostics ===');

  // Wait a bit for operations to settle
  sleep(5);

  const resourceCounts = collectResourceCounts(data.cookies, null);
  collectAPITimings(data.cookies, null);

  // Record final metrics
  diagnosticsUtil.metrics.forEach(({ key, gauge }) => {
    const count = resourceCounts[key]?.totalCount;
    if (count != null) {
      diagnosticsAfterGauge.add(count, { resource: key });
    }
  });

  console.log('Post-churn diagnostics collected');
}

function collectResourceCounts(cookies, tags) {
  console.log('Collecting resource counts...');
  const resourceCountsRaw = diagnosticsUtil.getLocalClusterResourceCounts(baseUrl, cookies);
  const resourceCounts = diagnosticsUtil.processResourceCounts(resourceCountsRaw);

  diagnosticsUtil.metrics.forEach(({ key, label, gauge }) => {
    const count = resourceCounts[key]?.totalCount;
    if (count != null) {
      gauge.add(count, tags);
      console.log(`Total ${label}: ${count}`);
    }
  });

  return resourceCounts;
}

function collectAPITimings(cookies, tags) {
  console.log('Collecting API response timings...');

  const timings = diagnosticsUtil.processResourceTimings(baseUrl, cookies);

  diagnosticsUtil.metrics.forEach(({ key, trend }) => {
    const time = timings[key];
    if (time != null) {
      trend.add(time, tags);
    }
  });

  const slowest = Object.entries(timings)
    .sort(([, a], [, b]) => b - a)
    .slice(0, 5);

  console.log('Top 5 slowest APIs:');
  slowest.forEach(([api, time], idx) => {
    console.log(`  ${idx + 1}. ${api}: ${time.toFixed(2)}ms`);
  });

  return timings;
}
