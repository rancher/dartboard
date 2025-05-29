import { check, fail, sleep } from 'k6';
import { getCookies, login } from "../rancher_utils.js";
import { Trend } from 'k6/metrics';
import * as crdUtil from "./crd_utils.js";
import { verifySchemaExistsPolling } from "./schema_utils.js"

const vus = __ENV.K6_VUS || 1
const perVuIterations = __ENV.PER_VU_ITERATIONS || 1
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD
const token = __ENV.TOKEN
const crdCount = __ENV.CRD_COUNT || 1
const namePrefix = "crontabs-test-"

export const epDataRecv = new Trend('endpoint_data_recv');
export const headerDataRecv = new Trend('header_data_recv');
export const timePolled = new Trend('time_polled', true);

export const options = {
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(95)', 'p(99)', 'count'],
  scenarios: {
    verify: {
      executor: 'per-vu-iterations',
      exec: 'verifySchemas',
      vus: vus,
      iterations: perVuIterations,
      maxDuration: '24h',
    }
  },
  thresholds: {
    http_req_failed: ['rate<=0.01'], // http errors should be less than 1%
    http_req_duration: ['p(99)<=500'], // 99% of requests should be below 500ms
    checks: ['rate>0.99'], // the rate of successful checks should be higher than 99%
    // If we want to only consider data points for a particular URL/endpoint we can filter by URL.
    header_data_recv: ['p(95) < 1024'],
    [`endpoint_data_recv{url:'/v1/schemas/<schemaID>'}`]: ['p(99) < 2048'], // bytes in this case
    [`endpoint_data_recv{url:'/v1/apiextensions.k8s.io.customresources/<CRD ID>'}`]: ['min > 2048'], // bytes in this case
    [`endpoint_data_recv{url:'/v1/apiextensions.k8s.io.customresources'}`]: ['min > 2048'], // bytes in this case
    [`endpoint_data_recv{url:'/v1/apiextensions.k8s.io.customresources/<CRD Name>'}`]: ['min > 2048'], // bytes in this case
    [`time_polled{url:'/v1/schemas/<schemaID>'}`]: ['p(99) < 500', `avg < 250`],
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
  // delete leftovers, if any
  let deleteAllFailed = cleanup(cookies, namePrefix)
  if (deleteAllFailed) fail("Failed to delete all existing crontab CRDs during setup!")
  let { _, crdArray } = crdUtil.getCRDsMatchingName(baseUrl, cookies, namePrefix)

  return { cookies: cookies, crdArray: generateCRDArray(cookies) }
}

export function generateCRDArray(cookies) {
  for (let i = 0; i < crdCount; i++) {
    let crdSuffix = `${i}`
    let res = crdUtil.createCRD(baseUrl, cookies, crdSuffix)
    crdUtil.trackDataMetricsPerURL(res, crdUtil.crdsTag, headerDataRecv, epDataRecv)
  }

  let crdArray = crdUtil.getCRDsMatchingNameVersions(baseUrl, cookies, namePrefix, 2)
  let expectedNumCRDs = crdArray.length
  if (expectedNumCRDs != crdCount) {
    fail(`Received (${expectedNumCRDs}) CRDs instead of # (${crdCount}) matching the expected content!`)
  }
  let finalCRD = crdArray[crdArray.length - 1]
  let schemaID = finalCRD.spec.group + "." + finalCRD.spec.names.singular
  let { res, timeSpent } = verifySchemaExistsPolling(baseUrl, cookies, schemaID, finalCRD.spec.versions[1].name, crdUtil.crdRefreshDelayMs * 5)
  crdUtil.trackDataMetricsPerURL(res, crdUtil.schemasTag, headerDataRecv, epDataRecv)
  console.log("TIME SPENT: ", timeSpent)
  timePolled.add(timeSpent, crdUtil.schemasTag)
  if (res.status != 200) {
    fail("Did not receive a HTTP 200 status on the final CRD's schemas.")
  }

  crdArray.forEach((crd, i) => {
    schemaID = crd.spec.group + "." + crd.spec.names.singular
    let { res, timeSpent } = verifySchemaExistsPolling(baseUrl, cookies, schemaID, crd.spec.versions[1].name, crdUtil.crdRefreshDelayMs * 5)
    crdUtil.trackDataMetricsPerURL(res, crdUtil.schemasTag, headerDataRecv, epDataRecv)
    console.log("TIME SPENT: ", timeSpent)
    timePolled.add(timeSpent, crdUtil.schemasTag)
    if (res.status != 200) {
      fail("Did not receive a HTTP 200 status on a CRD's schemas.")
    }
  })

  return crdUtil.getRandomArrayItems(crdArray, crdCount)
}

export function verifySchemas(data) {
  let existingIDs = [];
  let CRDs = data.crdArray
  let res = null;

  // add extra schema to crds
  let updated = 0
  CRDs.forEach((crd, i) => {
    console.log("VERSIONS LENGTH: ", crd.spec.versions.length)
    if (crd.spec.versions.length != 2) {
      fail("CRD DOES NOT HAVE EXPECTED # OF VERSIONS (2)")
    }
    res = crdUtil.getCRD(baseUrl, data.cookies, crd.id)
    let modifyCRD = JSON.parse(res.body)
    crdUtil.trackDataMetricsPerURL(res, crdUtil.crdTag, headerDataRecv, epDataRecv)

    modifyCRD.spec.versions[2] = newSchema
    // Unset previously stored version
    modifyCRD.spec.versions[1].storage = false
    console.log("MODIFIED VERSIONS LENGTH: ", modifyCRD.spec.versions.length)
    if (modifyCRD.spec.versions.length != 3) {
      fail("CRD DOES NOT HAVE EXPECTED # OF VERSIONS (3)")
    }

    let existingID = modifyCRD.spec.group + "." + modifyCRD.spec.names.singular
    existingIDs.push(existingID)
    res = crdUtil.updateCRD(baseUrl, data.cookies, modifyCRD)
    crdUtil.trackDataMetricsPerURL(res, crdUtil.putCRDTag, headerDataRecv, epDataRecv)
    updated += 1
  })

  existingIDs.forEach(id => {
    let { res, timeSpent } = verifySchemaExistsPolling(baseUrl, data.cookies, id, newSchema.name, crdUtil.crdRefreshDelayMs)
    let schemaBytes = res.body.length
    crdUtil.trackDataMetricsPerURL(res, crdUtil.schemasTag, headerDataRecv, epDataRecv)
    timePolled.add(timeSpent, crdUtil.schemasTag)
  })

  let reverted = 0
  // remove extra schema from crds

  CRDs.forEach((crd, i) => {
    // get latest version of each CRD
    let res = crdUtil.getCRD(baseUrl, data.cookies, crd.id)
    let updatedCRD = JSON.parse(res.body)
    crdUtil.trackDataMetricsPerURL(res, crdUtil.crdTag, headerDataRecv, epDataRecv)
    // swap out active versions
    updatedCRD.spec.versions[2].storage = false
    updatedCRD.spec.versions[2].served = false
    updatedCRD.spec.versions[1].storage = true

    res = crdUtil.updateCRD(baseUrl, data.cookies, updatedCRD)
    crdUtil.trackDataMetricsPerURL(res, crdUtil.putCRDTag, headerDataRecv, epDataRecv)
    sleep(crdUtil.crdRefreshDelaySeconds + 1)
    res = crdUtil.getCRD(baseUrl, data.cookies, crd.id)
    updatedCRD = JSON.parse(res.body)
    crdUtil.trackDataMetricsPerURL(res, crdUtil.crdTag, headerDataRecv, epDataRecv)
    updatedCRD.status.storedVersions.splice(1, 1)
    res = crdUtil.updateCRD(baseUrl, data.cookies, updatedCRD)
    crdUtil.trackDataMetricsPerURL(res, crdUtil.putCRDTag, headerDataRecv, epDataRecv)
    reverted += 1
  })

  existingIDs.forEach(id => {
    let { res, timeSpent } = verifySchemaExistsPolling(baseUrl, data.cookies, id, CRDs[0].spec.versions[1].name, crdUtil.crdRefreshDelayMs)
    let schemaBytes = res.body.length
    crdUtil.trackDataMetricsPerURL(res, crdUtil.schemasTag, headerDataRecv, epDataRecv)
    timePolled.add(timeSpent, crdUtil.schemasTag)
  })
}

export function teardown(data) {
  cleanup(data.cookies)
}
