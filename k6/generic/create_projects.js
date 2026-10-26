import { fail, sleep } from 'k6';
import exec from 'k6/execution';
import { Gauge } from 'k6/metrics';
import * as projectUtil from "../projects/project_utils.js";
import { getCookies, login } from "../rancher/rancher_utils.js";
import { getPrincipalIds, getCurrentUserPrincipalId, getClusterIds } from "../rancher/rancher_users_utils.js";
import { createPRTB, deletePRTBsByDescriptionLabel } from "../rbac/rbac_utils.js";
import { customHandleSummary } from './k6_utils.js';
import { stringify } from "../generic/generic_utils.js";

// Parameters
const prtbDescription = "Test-Project-PRTB"
const projectCount = Number(__ENV.PROJECT_COUNT)
const vus = 1
const customRoleTemplateBindingsPerProject = 5

// Option setting
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD

// Behavior setting
const createAttempts = __ENV.CREATE_ATTEMPTS ? Number(__ENV.CREATE_ATTEMPTS) : 30

export const handleSummary = customHandleSummary;

// Option setting
export const options = {
  insecureSkipTLSVerify: true,

  setupTimeout: '8h',

  scenarios: {
    createProjects: {
      executor: 'shared-iterations',
      exec: 'createProjects',
      vus: vus,
      iterations: projectCount,
      maxDuration: '1h',
    },
  },
  thresholds: {
    checks: ['rate>0.99']
  }
}

// Custom metrics
const projectsMetric = new Gauge('test_projects')

// Test functions, in order of execution

export function setup() {
  // log in
  if (!login(baseUrl, {}, username, password).status === 200) {
    fail(`could not login into cluster`)
  }
  const cookies = getCookies(baseUrl)

  // delete leftovers, if any
  cleanup(cookies)
  // return data that remains constant throughout the test
  return {
    cookies: cookies,
    principalIds: getPrincipalIds(baseUrl, cookies),
    myId: getCurrentUserPrincipalId(baseUrl, cookies),
    clusterIds: getClusterIds(baseUrl, cookies)
  }
}

function cleanup(cookies) {
  deletePRTBsByDescriptionLabel(baseUrl, cookies, prtbDescription)
  projectUtil.getProjects(baseUrl, cookies)
  let { projectArray } = projectUtil.getNormanProjectsMatchingName(baseUrl, cookies, "Test ")
  console.log(`Found ${projectArray.length} projects to clean up`)
  projectArray.forEach(r => {
    projectUtil.deleteNormanProject(baseUrl, cookies, r["id"])
    sleep(0.5)
  })
}

const mainRoleTemplateIds = ["project-owner", "project-member", "read-only", "custom"]
const customRoleTemplateIds = [
  "create-ns", "configmaps-manage", "ingress-manage", "projectroletemplatebindings-manage",
  "secrets-manage", "serviceaccounts-manage", "services-manage", "persistentvolumeclaims-manage",
  "workloads-manage", "configmaps-view", "ingress-view", "monitoring-ui-view", "projectroletemplatebindings-view",
  "secrets-view", "serviceaccounts-view", "services-view", "persistentvolumeclaims-view", "workloads-view"
]

export function createProjects(data) {
  let response
  const i = exec.scenario.iterationInTest
  const cookies = data.cookies
  const myId = data.myId
  const clusterId = data.clusterIds[i % data.clusterIds.length]

  const projectBody = JSON.stringify({
    "type": "project",
    "name": `Test Project ${i}`,
    "description": `Test Project ${i}`,
    "annotations": {},
    "labels": {},
    "clusterId": clusterId,
    "creatorId": `local://${myId}`,
    "containerDefaultResourceLimit": {
      "limitsCpu": "4m",
      "limitsMemory": "5Mi",
      "requestsCpu": "2m",
      "limitsGpu": 6,
      "requestsMemory": "3Mi"
    },
    "resourceQuota": {
      "limit": {
        "configMaps": "9",
        "limitsMemory": "900Mi",
        "limitsCpu": "90m",
        "persistentVolumeClaims": "9000"
      }
    },
    "namespaceDefaultResourceQuota": {
      "limit": {
        "configMaps": "6",
        "limitsMemory": "600Mi",
        "limitsCpu": "60m",
        "persistentVolumeClaims": "6000"
      }
    }
  })

  response = projectUtil.createNormanProject(baseUrl, cookies, projectBody)

  const projectId = JSON.parse(response.body)["id"]

  const principalId = data.principalIds[i % data.principalIds.length]
  const userId = principalId.startsWith('local://') ? principalId.replace('local://', '') : principalId
  const mainRoleTemplateId = mainRoleTemplateIds[i % mainRoleTemplateIds.length]
  const roleTemplateIds = mainRoleTemplateId !== "custom" ? [mainRoleTemplateId] : Array.from({ length: customRoleTemplateBindingsPerProject }, (_, j) => (
    customRoleTemplateIds[(i + j) % customRoleTemplateIds.length]
  ))

  for (const roleTemplateId of roleTemplateIds) {

    // HACK: creating projectroletemplatebindings might fail with 404 if the project controller is too slow
    // allow up to 30 retries
    let success = false
    let alreadyExists = false
    for (let j = 0; j < createAttempts && !success; j++) {
      response = createPRTB(baseUrl, cookies, prtbDescription, projectId, roleTemplateId, userId)

      success = response.status === 201
      alreadyExists = response.status === 409
      if (!success && !alreadyExists) {
        console.log(`[Attempt ${j + 1}/${createAttempts}] Failed to create projectroletemplatebinding (status: ${response.status}).`)
        let retry = j !== createAttempts - 1 && createAttempts > 1
        if (retry) {
          console.log(`Retrying... Error: ${response.body}`)
        } else {
          console.log(`Reached maximum attempts (${createAttempts}) for creating projectroletemplatebinding. Error: ${response.body}`)
        }
        sleep(Math.random())
      }
      if (alreadyExists) {
        console.log(`[Attempt ${j + 1}/${createAttempts}] ${stringify(JSON.parse(response.body).message)}. Skipping...`)
        sleep(Math.random())
        break
      }
    }
    if (!success && !alreadyExists) {
      fail(`/v3/projectroletemplatebindings did not return 201 after ${createAttempts} attempts`)
    }
  }

  projectsMetric.add(projectCount)
}
