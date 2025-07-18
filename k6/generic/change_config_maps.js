import { sleep, check, fail } from 'k6';
import encoding from 'k6/encoding';
import exec from 'k6/execution';
import * as k8s from './k8s.js'
import * as diagnosticsUtil from "../rancher/rancher_diagnostics.js";
import { getCookies, login, retryUntilOneOf } from "../rancher/rancher_utils.js";
import http from 'k6/http';
import { Gauge, Trend, Counter } from 'k6/metrics';
import * as userUtil from "../rancher/rancher_users_utils.js"
import * as namespacesUtil from "../namespaces/namespace_utils.js"
import * as projectsUtil from "../projects/project_utils.js"

// Parameters
const namespace = "dartboard-test"
const configMapData = encoding.b64encode("a".repeat(1))
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD
const token = __ENV.TOKEN
const cluster = __ENV.CLUSTER || "local"
const resource = __ENV.RESOURCE || "configmaps"
const api = __ENV.API || "steve"
// Pagination options
const paginationStyle = __ENV.PAGINATION_STYLE || "k8s"
const pageSize = parseInt(__ENV.PAGE_SIZE || 100)
const firstPageOnly = __ENV.FIRST_PAGE_ONLY === "true"
const urlSuffix = __ENV.URL_SUFFIX || ""
const pauseSeconds = parseFloat(__ENV.PAUSE_SECONDS || 5.0)

// Option setting
const vus = Number(__ENV.VUS || 5)
const duration = '2h'
const diagnosticsInterval = (__ENV.DIAGNOSTICS_INTERVAL || "20m"); // # time unit
// 2 requests per iteration (for change() func), so iteration rate is 1/2 of request rate
const changeIPS = (__ENV.TARGET_RPS || 10) / 2
// 1 request per iteration (for list() func)
const listIPS = 1
const kubeconfig = k8s.kubeconfig(__ENV.KUBECONFIG, __ENV.CONTEXT)

// Metrics
const diagnosticsBeforeGauge = new Gauge('diagnostics_before_churn');
const diagnosticsDuringTrend = new Trend('diagnostics_during_churn');
const diagnosticsAfterGauge = new Gauge('diagnostics_after_churn');
const churnOpsCounter = new Counter('churn_operations_total');
const churnOpsRate = new Trend('churn_operations_rate');
let changeEvents = 0

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
    preChurnDiagnostics: {
      executor: 'shared-iterations',
      exec: 'preChurnDiagnostics',
      vus: 1,
      iterations: 1,
      startTime: '0s',
      maxDuration: '5m',
    },
    change: {
      executor: 'constant-arrival-rate',
      exec: 'change',
      preAllocatedVUs: vus,
      duration: duration,
      rate: changeIPS,
      maxVUs: 20,
      startTime: '5m'
    },
    list: {
      executor: 'constant-arrival-rate',
      exec: 'list',
      preAllocatedVUs: vus,
      duration: duration,
      rate: listIPS,
      maxVUs: 20,
      startTime: '5m'
    },
    duringChurnDiagnostics: {
      executor: 'constant-arrival-rate',
      exec: 'duringChurnDiagnostics',
      preAllocatedVUs: 1,
      rate: 1, // Every N seconds
      timeUnit: diagnosticsInterval,
      duration: duration,
      startTime: '5m',
    },
    postChurnDiagnostics: {
      executor: 'shared-iterations',
      exec: 'postChurnDiagnostics',
      vus: 1,
      iterations: 1,
      startTime: `${parseInt(duration) + 5}m`,
      maxDuration: '5m',
    },
  },
  thresholds: {
    http_req_failed: ['rate<=0.05'], // Allow 5% error rate during churn
    http_req_duration: ['p(95)<=2000'], // 95% of requests should be below 2s
    checks: ['rate>0.95'], // 95% success rate
    churn_operations_total: [`count>=${changeIPS * parseInt(duration) * 0.9}`], // At least 90% of target ops
    churn_operations_rate: ['med>=100']
  }
};

// Simulate a pause after a click - on average pauseSeconds, +/- a random quantity up to 50%
function pause() {
  sleep(pauseSeconds + (Math.random() - 0.5) * 2 * pauseSeconds / 2)
}

// const pauseTime = pause()

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
  // log in
  // if session cookie was specified, save it
  if (token) {
    return { R_SESS: token }
  }

  var cookies = {}

  let adminLoginRes = login(baseUrl, {}, username, password)

  if (adminLoginRes.status !== 200) {
    fail(`could not login to cluster as admin`)
  }
  cookies = getCookies(baseUrl)

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

  return cookies
}

export function preChurnDiagnostics(cookies) {
  console.log('=== Collecting PRE-CHURN diagnostics ===');

  const resourceCounts = collectResourceCounts(cookies, null);
  const apiTimings = collectAPITimings(cookies, null);

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
  changeEvents += 1
  k8s.del(`${kubeconfig.url}/api/v1/namespaces/${namespace}/configmaps/${name}`)
  churnOpsCounter.add(1, { resource: resource });
  changeEvents += 1
}

export function list(cookies) {
  if (api === "steve") {
    const url = cluster === "local" ?
      `${baseUrl}/v1/${resource}` :
      `${baseUrl}/k8s/clusters/${cluster}/v1/${resource}`

    if (paginationStyle === "k8s") {
      listWithK8sStylePagination(url, cookies)
    }
    else if (paginationStyle === "steve") {
      listWithSteveStylePagination(url, cookies)
    }
    else {
      fail("Invalid PAGINATION_STYLE value: " + paginationStyle)
    }
  }
  else if (api === "norman") {
    const url = `${baseUrl}/v3/${resource}`
    listWithNormanStylePagination(url, cookies)
  }
  else {
    fail("Invalid API value: " + api)
  }
}

function listWithK8sStylePagination(url, cookies) {
  let revision = null
  let continueToken = null
  while (true) {
    const fullUrl = url + "?limit=" + pageSize +
      (revision != null ? "&revision=" + revision : "") +
      (continueToken != null ? "&continue=" + continueToken : "") +
      urlSuffix

    const res = http.get(fullUrl, { cookies: cookies })

    const criteria = {}
    criteria[`listing ${resource} from cluster ${cluster} (steve with k8s style pagination) succeeds`] = (r) => r.status === 200
    criteria[`no slow pagination errors (410 Gone) detected`] = (r) => r.status !== 410
    check(res, criteria)

    try {
      const body = JSON.parse(res.body)
      if (body === undefined || body.continue === undefined || firstPageOnly) {
        break
      }
      if (revision == null) {
        revision = body.revision
      }
      continueToken = body.continue
    } catch (e) {
      if (e instanceof SyntaxError) {
        fail("Response body does not parse as JSON: " + res.body)
      }
      throw e
    }

    pause()
  }
}

function listWithSteveStylePagination(url, cookies) {
  let i = 1
  let revision = null
  while (true) {
    const fullUrl = url + "?pagesize=" + pageSize + "&page=" + i +
      (revision != null ? "&revision=" + revision : "") +
      urlSuffix

    const res = http.get(fullUrl, { cookies: cookies })

    const criteria = {}
    criteria[`listing ${resource} from cluster ${cluster} (steve style pagination) succeeds`] = (r) => r.status === 200
    criteria[`no slow pagination errors (410 Gone) detected`] = (r) => r.status !== 410
    check(res, criteria)

    try {
      const body = JSON.parse(res.body)
      if (body === undefined || body.data === undefined || body.length === 0 || firstPageOnly) {
        break
      }
      if (revision == null) {
        revision = body.revision
      }
      i = i + 1
    }
    catch (e) {
      if (e instanceof SyntaxError) {
        fail("Response body does not parse as JSON: " + res.body)
      }
      throw e
    }

    pause()
  }
}

function listWithNormanStylePagination(url, cookies) {
  let nextUrl = url + "?limit=" + pageSize
  while (true) {
    const res = http.get(nextUrl, { cookies: cookies })

    const criteria = {}
    criteria[`listing ${resource} from cluster ${cluster} (norman style pagination) succeeds`] = (r) => r.status === 200
    criteria[`no slow pagination errors (410 Gone) detected`] = (r) => r.status !== 410
    check(res, criteria)

    try {
      const body = JSON.parse(res.body)
      if (body === undefined || body.pagination === undefined || body.pagination.partial === undefined || body.pagination.next === undefined) {
        break
      }
      nextUrl = body.pagination.next
    } catch (e) {
      if (e instanceof SyntaxError) {
        fail("Response body does not parse as JSON: " + res.body)
      }
      throw e
    }

    pause()
  }
}

// export function eventsPerSecond() {
//   // Change Events per second
//   console.log("# EVENTS")
//   console.log(changeEvents)
//   const currentRate = changeEvents / ((Date.now() - exec.test.startTime) / 1000);
//   console.log("EVENTS/SECOND")
//   console.log(Math.round(currentRate))
//   churnOpsRate.add(Math.round(currentRate));
// }

export function duringChurnDiagnostics(cookies) {
  console.log('=== Collecting DURING-CHURN diagnostics ===');

  const start = Date.now();
  const resourceCounts = collectResourceCounts(cookies, null);
  const apiTimings = collectAPITimings(cookies, null);
  const duration = Date.now() - start;

  // Record timing for diagnostic collection overhead
  diagnosticsDuringTrend.add(duration);

  console.log(`During-churn diagnostics collected in ${duration}ms, current rate: ${currentRate.toFixed(2)} ops/s`);
}

export function postChurnDiagnostics(cookies) {
  console.log('=== Collecting POST-CHURN diagnostics ===');

  // Wait a bit for operations to settle
  sleep(5);

  const resourceCounts = collectResourceCounts(cookies, null);
  const apiTimings = collectAPITimings(cookies, null);

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
