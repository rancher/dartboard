import encoding from 'k6/encoding';
import exec from 'k6/execution';
import * as diagnosticsUtil from "./rancher_diagnostics.js";
import { getCookies, login, logout } from "./rancher_utils.js";
import { check, fail, sleep } from 'k6';
import * as k8s from '../generic/k8s.js'
import http from 'k6/http';


// Parameters
const iterations = Number(__ENV.ITERATIONS || 1)
const vus = Number(__ENV.VUS || 1)

// Option setting
const kubeconfig = k8s.kubeconfig(__ENV.KUBECONFIG, __ENV.CONTEXT)
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD

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
    collectDiagnostics: {
      executor: 'shared-iterations',
      exec: 'collectDiagnostics',
      vus: vus,
      iterations: iterations,
      maxDuration: '1h',
    },

  },
  thresholds: {
    http_req_failed: ['rate<=0.01'], // http errors should be less than 1%
    http_req_duration: ['p(99)<=500'], // 99% of requests should be below 500ms
    checks: ['rate>0.99'], // the rate of successful checks should be higher than 99%
    [`api_event_duration`]: ['avg<1000', 'p(95)<2000'],
    [`api_k8sevent_duration`]: ['avg<1000', 'p(95)<2000'],
    [`api_setting_duration`]: ['avg<1000', 'p(95)<2000'],
    [`api_clusterrole_duration`]: ['avg<1000', 'p(95)<2000'],
    [`api_crd_duration`]: ['avg<1000', 'p(95)<2000'],
    [`api_role_duration`]: ['avg<1000', 'p(95)<2000'],
    [`api_rolebinding_duration`]: ['avg<1000', 'p(95)<2000'],
    [`api_clusterrolebinding_duration`]: ['avg<1000', 'p(95)<2000'],
    [`api_globalrolebinding_duration`]: ['avg<1000', 'p(95)<2000'],
    [`api_configmap_duration`]: ['avg<1000', 'p(95)<2000'],
    [`api_serviceaccount_duration`]: ['avg<1000', 'p(95)<2000'],
    [`api_secret_duration`]: ['avg<1000', 'p(95)<2000'],
    [`api_pod_duration`]: ['avg<4000', 'p(95)<2000'],
    [`api_deployment_duration`]: ['avg<4000', 'p(95)<2000'],
    [`api_service_duration`]: ['avg<4000', 'p(95)<2000'],
    [`api_apiservice_duration`]: ['avg<4000', 'p(95)<2000'],
    [`api_roletemplate_duration`]: ['avg<4000', 'p(95)<2000'],
    [`api_project_duration`]: ['avg<1000', 'p(95)<2000'],
    [`api_namespace_duration`]: ['avg<1000', 'p(95)<2000'],
  }
};

export function setup() {
  // log in
  var cookies = {}

  let adminLoginRes = login(baseUrl, {}, username, password)

  if (adminLoginRes.status !== 200) {
    fail(`could not login to cluster as admin`)
  }
  cookies = getCookies(baseUrl)

  return cookies
}

export function collectDiagnostics(cookies) {
  collectResourceCounts(cookies, null)
  collectAPITimings(cookies, null)
}

function collectResourceCounts(cookies, tags) {
  console.log('Collecting resource counts:');

  let resourceCountsRaw = diagnosticsUtil.getLocalClusterResourceCounts(baseUrl, cookies)
  let resourceCounts = diagnosticsUtil.processResourceCounts(resourceCountsRaw)

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
  console.log(`Collecting API response timings:`);
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
