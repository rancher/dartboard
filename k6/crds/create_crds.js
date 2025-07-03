import { check, fail, sleep } from 'k6';
import http from 'k6/http'
import { Trend } from 'k6/metrics';
import { getCookies, login } from "../rancher/rancher_utils.js";
import exec from "k6/execution";
import { randomString } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import * as crdUtil from "./crd_utils.js";
import * as k6Util from "../generic/k6_utils.js";

const vus = __ENV.K6_VUS || 20
const crdCount = __ENV.CRD_COUNT || 500
const iterations = __ENV.PER_VU_ITERATIONS || 30
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD
const namePrefix = "crontabs-test-"

export const epDataRecv = new Trend('endpoint_data_recv');
export const headerDataRecv = new Trend('header_data_recv');
export const timePolled = new Trend('time_polled', true);

export const options = {
  scenarios: {
    create: {
      executor: 'shared-iterations',
      exec: 'createCRDs',
      vus: vus,
      iterations: iterations,
      maxDuration: '24h',
    }
  },
  thresholds: {
    http_req_failed: ['rate<=0.01'], // http errors should be less than 1%
    http_req_duration: ['p(99)<=500'], // 95% of requests should be below 500ms
    checks: ['rate>0.99'], // the rate of successful checks should be higher than 99%
    header_data_recv: ['p(95) < 1024'],
    [`endpoint_data_recv{url:'/v1/apiextensions.k8s.io.customresourcedefinitions'}`]: ['min > 2048'], // bytes in this case
    [`time_polled{url:'/v1/apiextensions.k8s.io.customresourcedefinitions'}`]: ['p(99) < 5000', 'avg < 2500'],
  },
  setupTimeout: '15m',
  teardownTimeout: '15m'
}

function cleanup(cookies, namePrefix) {
  let { _, crdArray } = crdUtil.getCRDsMatchingName(baseUrl, cookies, namePrefix)
  let deleteAllFailed = false
  crdArray.forEach(r => {
    let delRes = crdUtil.deleteCRD(baseUrl, cookies, r["id"])
    if (delRes.status !== 200 && delRes.status !== 204) deleteAllFailed = true
    sleep(0.5)
  })
  return deleteAllFailed
}

// Test functions, in order of execution
export function setup() {
  // log in
  if (login(baseUrl, {}, username, password).status !== 200) {
    fail(`could not login into cluster`)
  }
  const cookies = getCookies(baseUrl)

  // delete leftovers, if any
  let deleteAllFailed = cleanup(cookies, namePrefix)
  if (deleteAllFailed) fail("Failed to delete all existing crontab CRDs during setup!")
  // return data that remains constant throughout the test
  return cookies
}

export function createCRDs(cookies) {
  for (let i = 0; i < crdCount; i++) {
    let crdSuffix = `${exec.vu.idInTest}-${randomString(4)}`
    let res = crdUtil.createCRD(baseUrl, cookies, crdSuffix)
    k6Util.trackResponseSizePerURL(res, crdUtil.crdsTag, headerDataRecv, epDataRecv)
    sleep(0.5)
  }
  sleep(0.15)
  let { _, timeSpent } = crdUtil.verifyCRDs(baseUrl, cookies, namePrefix, 500, crdUtil.crdRefreshDelayMs * 5)
  timePolled.add(timeSpent, crdUtil.crdsTag)
  sleep(60)
  let deleteAllFailed = cleanup(cookies, namePrefix)
  if (deleteAllFailed) fail("Failed to delete all existing crontab CRDs!")
  // Give time for resource usage to cool down between iterations
  sleep(900)
}
