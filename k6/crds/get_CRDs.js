import http from 'k6/http'
import { check, fail, sleep } from 'k6';
import exec from 'k6/execution';
import { Trend } from 'k6/metrics';
import { getCookies, login } from "../rancher_utils.js";
import * as k8s from '../k8s.js'
import * as crdUtil from "./crd_utils.js";


const vus = __ENV.K6_VUS || 20
const crdCount = __ENV.CRD_COUNT || 500
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD

export const epDataRecv = new Trend('endpoint_data_recv');
export const headerDataRecv = new Trend('header_data_recv');

// Option setting
export const options = {
  setupTimeout: '8h',
  scenarios: {
    get: {
      executor: 'per-vu-iterations',
      exec: 'getCRDs',
      vus: vus,
      iterations: crdCount,
      maxDuration: '1h',
    }
  },
  thresholds: {
    http_req_failed: ['rate<=0.01'], // http errors should be less than 1%
    http_req_duration: ['p(99)<=500'], // 95% of requests should be below 500ms
    checks: ['rate>0.99'], // the rate of successful checks should be higher than 99%
    header_data_recv: ['p(95) < 1024'],
    [`endpoint_data_recv{url:'/v1/apiextensions.k8s.io.customresourcedefinitions'}`]: ['min > 2048'], // bytes in this case
    [`time_polled{url:'/v1/apiextensions.k8s.io.customresourcedefinitions'}`]: ['p(99) < 5000', 'avg < 2500'],
  }
}

export function setup() {
  // log in
  if (!login(baseUrl, {}, username, password)) {
    fail(`could not login into cluster`)
  }
  const cookies = getCookies(baseUrl)

  let { _, crdArray } = crdUtil.getCRDsMatchingName(baseUrl, cookies, namePrefix)

  // return data that remains constant throughout the test
  return { cookies: cookies, crdArray: checkAndBuildCRDArray(cookies, crdArray) }
}

export function checkAndBuildCRDArray(cookies, crdArray) {
  let retries = 3
  let attempts = 0
  while (crdArray.length != crdCount && attempts < retries) {
    console.log("Creating needed CRDs")
    // delete leftovers, if any so that we create exactly crdCount
    if (crdArray.length == crdCount) {
      console.log("Finished setting up expected CRD count")
      break;
    }
    if (crdArray.length > 0) {
      let deleteAllFailed = cleanup(cookies,)
      if (deleteAllFailed && attempts == (retries - 1)) fail("Failed to delete all existing crontab CRDs during setup!")
    }
    for (let i = 0; i < crdCount; i++) {
      let crdSuffix = `${i}`
      let res = crdUtil.createCRD(baseUrl, cookies, crdSuffix)
      crdUtil.trackDataMetricsPerURL(res, crdUtil.crdsTag, headerDataRecv, epDataRecv)
      sleep(0.25)
    }
    let { res, crdArray } = crdUtil.getCRDsMatchingName(baseUrl, cookies, namePrefix)
    if (res.status != 200 && attempts == (retries - 1)) fail("Failed to retrieve expected CRDs during setup")
    attempts += 1
  }
  if (crdArray.length != crdCount) fail("Failed to create expected # of CRDs")
  console.log("Expected number of CRDs accounted for ", crdArray.length)
  return crdArray
}

export function getCRDs(data) {
  let res = crdUtil.getCRDs(baseUrl, data.cookies)
  crdUtil.trackDataMetricsPerURL(res, crdUtil.crdsTag, headerDataRecv, epDataRecv)
  return res
}
