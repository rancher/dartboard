import { check, fail } from 'k6';
import { sleep } from 'k6';
import exec from 'k6/execution';
import http from 'k6/http';
import { customHandleSummary } from './generic_utils.js';

// Creates dummy pods via Steve

// Parameters
const namespace = __ENV.NAMESPACE || "scalability-test"
const podCount = Number(__ENV.POD_COUNT)
const vus = __ENV.VUS || 1
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD
const token = __ENV.TOKEN

export const handleSummary = customHandleSummary;

// Option setting
export const options = {
  insecureSkipTLSVerify: true,

  scenarios: {
    createPods: {
      executor: 'shared-iterations',
      exec: 'createPods',
      vus: vus,
      iterations: podCount,
      maxDuration: '1h',
    },
  },
  thresholds: {
    checks: ['rate>0.99']
  }
}

// Test functions, in order of execution
export function setup() {
  var cookies = {}
  // if session cookie was specified, save it
  if (token) {
    cookies = { R_SESS: token }
  }
  // if credentials were specified, log in
  else if (username && password) {
    const res = http.post(`${baseUrl}/v3-public/localProviders/local?action=login`, JSON.stringify({
      "description": "UI session",
      "responseType": "cookie",
      "username": username,
      "password": password
    }))

    check(res, {
      'logging in returns status 200': (r) => r.status === 200,
    })

    cookies = http.cookieJar().cookiesForURL(res.url)
  }
  else {
    fail("Please specify either USERNAME and PASSWORD or TOKEN")
  }

  // delete leftovers, if any
  var res = http.del(`${baseUrl}/v1/namespaces/${namespace}`, { cookies: cookies })
  check(res, {
    'DELETE returns status 200 or 404': (r) => r.status === 200 || r.status === 404,
  })

  // create empty namespace
  const body = {
    "metadata": {
      "name": namespace,
    },
  }
  create(`${baseUrl}/v1/namespaces`, body, { cookies: cookies })

  return cookies
}

function create(url, body, headers) {
  const res = http.post(url, JSON.stringify(body), headers)
  check(res, {
    'POST returns status 201 or 409': (r) => r.status === 201 || r.status === 409,
  })

  if (res.status === 409) {
    // wait a bit and try again
    sleep(Math.random())

    return create(url, body)
  }
}

export function createPods(cookies) {
  var body = {
    "metadata": {
      "namespace": namespace,
      "labels": {
        "workload.user.cattle.io/workloadselector": "pod-test-namespace-t1"
      },
      "name": `test-pod-${exec.scenario.iterationInTest}`,
      "annotations": {}
    },
    "spec": {
      "containers": [
        {
          "imagePullPolicy": "Always",
          "name": "container-0",
          "volumeMounts": [],
          "image": "nginx:alpine"
        }
      ],
      "initContainers": [],
      "imagePullSecrets": [],
      "volumes": [],
      "affinity": {}
    }
  }
  create(`${baseUrl}/v1/pods`, body, { cookies: cookies })
}

export function teardown(data) {
  // delete leftovers, if any
  var res = http.del(`${baseUrl}/v1/namespaces/${namespace}`, { cookies: data })
  check(res, {
    'DELETE returns status 200 or 404': (r) => r.status === 200 || r.status === 404,
  })
}
