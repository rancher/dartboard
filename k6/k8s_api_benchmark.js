import * as k8s from './generic/k8s.js';
import { customHandleSummary } from './generic/k6_utils.js';

// Parameters
const vus = __ENV.VUS || 1;
const perVuIterations = __ENV.PER_VU_ITERATIONS || 30;
const resource = __ENV.RESOURCE || 'configmaps';
const limit = __ENV.LIMIT || 5000;
const namespace = __ENV.NAMESPACE || 'scalability-test';
const baseUrl = __ENV.BASE_URL || kubeconfig.url;

export const handleSummary = customHandleSummary;

// Option setting
export const options = {
  insecureSkipTLSVerify: true,

  tlsAuth: [
    {
      cert: k8s.kubeconfig.cert,
      key: k8s.kubeconfig.key,
    },
  ],

  scenarios: {
    list: {
      executor: 'per-vu-iterations',
      exec: 'list',
      vus: vus,
      iterations: perVuIterations,
      maxDuration: '24h',
    },
  },
  thresholds: {
    checks: ['rate>0.99'],
  },
};

// Test functions, in order of execution

export function list() {
  k8s.list(`${baseUrl}/api/v1/namespaces/${namespace}/${resource}`, limit);
}
