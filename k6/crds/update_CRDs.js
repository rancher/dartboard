import { check, fail, sleep } from 'k6';
import http from 'k6/http'
import { Trend } from 'k6/metrics';
import { getCookies, login } from "./rancher_utils.js";
import { vu as metaVU } from 'k6/execution'
import * as crdUtil from "./crd_utils.js";


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
    update: {
      executor: 'shared-iterations',
      exec: 'updateCRDs',
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
    [`endpoint_data_recv{url:'/v1/apiextensions.k8s.io.customresourcedefinitions/<CRD ID>'}`]: ['max < 512'], // bytes in this case
    [`time_polled{url:'/v1/apiextensions.k8s.io.customresourcedefinitions'}`]: ['p(99) < 5000', 'avg < 2500'],
  },
  setupTimeout: '30m',
  teardownTimeout: '30m'
}

let newSchema = {
  "name": "v3",
  "schema": {
    "openAPIV3Schema": {
      "properties": {
        "spec": {
          "properties": {
            "cronSpec": {
              "type": "string"
            },
            "image": {
              "type": "string"
            },
            "newField": {
              "type": "string"
            },
            "replicas": {
              "type": "integer"
            }
          },
          "type": "object"
        }
      },
      "type": "object"
    }
  },
  "served": true,
  "storage": true,
}


function cleanup(cookies) {
  let { res, crdArray } = crdUtil.getCRDsMatchingName(baseUrl, cookies, namePrefix)
  console.log("Got CRD Status in delete: ", res.status)
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
  if (!login(baseUrl, {}, username, password)) {
    fail(`could not login into cluster`)
  }
  const cookies = getCookies(baseUrl)

  let deleteAllFailed = cleanup(cookies, namePrefix)
  if (deleteAllFailed) fail("Failed to delete all existing crontab CRDs during setup!")
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
    let { res, crdArray: crds } = crdUtil.getCRDsMatchingName(baseUrl, cookies, namePrefix)
    if (Array.isArray(crds) && crds.length) crdArray = crds
    if (res.status != 200 && attempts == (retries - 1)) fail("Failed to retrieve expected CRDs during setup")
    attempts += 1
  }
  if (crdArray.length != crdCount) fail("Failed to create expected # of CRDs")
  console.log("Expected number of CRDs accounted for ", crdArray.length)
  sleep(300)
  return crdArray
}

export function updateCRDs(data) {
  let crds = data.crdArray
  crds.forEach(c => {
    if (c.spec.versions.length != 2) {
      fail("CRD DOES NOT HAVE EXPECTED # OF VERSIONS (2)")
    }
    let res = crdUtil.getCRD(baseUrl, data.cookies, c.id)
    let modifyCRD = JSON.parse(res.body)
    crdUtil.trackDataMetricsPerURL(res, crdUtil.crdTag, headerDataRecv, epDataRecv)

    modifyCRD.spec.versions[1].storage = false
    modifyCRD.spec.versions[2] = newSchema
    if (modifyCRD.spec.versions.length != 3) {
      fail("CRD DOES NOT HAVE EXPECTED # OF VERSIONS (3)")
    }
    res = crdUtil.updateCRD(baseUrl, data.cookies, modifyCRD)
    crdUtil.trackDataMetricsPerURL(res, crdUtil.putCRDTag, headerDataRecv, epDataRecv)
    sleep(0.25)
  })
  let { res, timeSpent } = crdUtil.verifyCRDs(baseUrl, data.cookies, namePrefix, crdCount, crdUtil.crdRefreshDelayMs * 5)
  timePolled.add(timeSpent, crdUtil.crdsTag)
  sleep(0.15)
  let numUpdated = JSON.parse(res.body)["data"].filter(r => r["metadata"]["name"].startsWith(namePrefix)).filter(r => r.spec.versions.length == 3).length
  check((numUpdated / crdCount), {
    'Total % of CRDs reflecting the newly added version >= 99%': (v) => v >= 0.99,
  })
}
