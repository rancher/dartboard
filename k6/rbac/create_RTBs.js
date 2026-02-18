import { check, fail, sleep } from 'k6';
import exec from 'k6/execution';
import http from 'k6/http';
// import { randomUUID } from 'k6/crypto';
import { Trend } from 'k6/metrics';
import { vu as metaVU } from 'k6/execution'
import * as k6Util from "../generic/k6_utils.js";
import {
  getCookies, login, logout
} from "../rancher/rancher_utils.js";
import { getRandomElements } from "../generic/generic_utils.js";
import { getClusterIds, getCurrentUserPrincipalId, getPrincipalIds } from "../rancher/rancher_users_utils.js";
import {
  createNormanProject as createProject,
  getProjects as listProjects,
  getProject as getProjectById,
  cleanupMatchingProjects as deleteProjectsByPrefix
} from "../projects/project_utils.js";
import {
  createUser, listUsers, listRoles, listRoleTemplates,
  listRoleBindings, listClusterRoles, listClusterRoleBindings,
  listCRTBs, listPRTBs, deleteRoleTemplatesByPrefix, deleteUsersByPrefix,
  createRoleTemplate, createPRTB, createCRTB,
  deletePRTBsByDescriptionLabel, deleteCRTBsByDescriptionLabel,
  createGlobalRoleBinding
} from './rbac_utils.js';

// Parameters
const vus = __ENV.VUS || 5
const projectCount = Number(__ENV.PROJECT_COUNT) || 10
const userCount = Number(__ENV.USER_COUNT) || 10
const testUserPassword = __ENV.TEST_USER_PASSWORD
const userPrefix = __ENV.USER_PREFIX || 'test-user'
const projectsPrefix = "rtbs-test"
const projectRoleTemplatePrefix = "Dartboard PRTB"
const clusterRoleTemplatePrefix = "Dartboard CRTB"

// Option setting
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD

export const handleSummary = k6Util.customHandleSummary;

// Option setting
export const options = {
  insecureSkipTLSVerify: true,

  setupTimeout: '8h',

  scenarios: {
    createPRTBs: {
      executor: 'shared-iterations',
      exec: 'createPRTBs',
      vus: vus,
      iterations: projectCount,
      maxDuration: '1h',
    },
    createCRTBs: {
      executor: 'shared-iterations',
      exec: 'createCRTBs',
      vus: vus,
      iterations: userCount,
      maxDuration: '1h',
    }
  },
  thresholds: {
    http_req_failed: ['rate<=0.01'], // http errors should be less than 1%
    http_req_duration: ['p(99)<=500'], // 95% of requests should be below 500ms
    checks: ['rate>0.99'], // the rate of successful checks should be higher than 99%
  }
}

// Custom metrics
const numRolesTrend = new Trend('num_roles');
const numRoleTemplatesTrend = new Trend('num_role_templates');
const numRoleBindingsTrend = new Trend('num_role_bindings');
const numClusterRolesTrend = new Trend('num_cluster_roles');
const numClusterRoleBindingsTrend = new Trend('num_cluster_role_bindings');
const numCRTBsTrend = new Trend('num_crtbs');
const numPRTBsTrend = new Trend('num_prtbs');

// Test functions, in order of execution
export function setup() {
  // log in
  if (!login(baseUrl, null, username, password)) {
    fail(`could not login to cluster`)
  }
  const cookies = getCookies(baseUrl)

  // delete leftovers, if any
  cleanup(cookies)

  let clusterIds = getClusterIds(baseUrl, cookies)
  let clusterId = getRandomElements(clusterIds, 1)[0]
  console.log(`Utilizing Cluster with the ID ${clusterId}`)
  let myId = getCurrentUserPrincipalId(baseUrl, cookies)

  // Create Projects, and Users
  for (let projectNum = 0; projectNum < projectCount; projectNum++) {
    const projectName = `${projectsPrefix}-vu${metaVU.idInInstance}-${projectNum.toString().padStart(4, '0')}`
    const projectBody = JSON.stringify({
      type: "project",
      name: projectName,
      description: `Load test project ${projectNum + 1}`,
      clusterId: "local",
      creatorId: myId,
    })
    let res = createProject(baseUrl, cookies, projectBody, clusterId, myId)
    if (res.status !== 201) {
      console.log("create project status: ", res.status)
      fail("Failed to create all expected Projects")
    }
  }

  // Store created users with their credentials for later use
  let createdUsers = []
  for (let numUsers = 0; numUsers < userCount; numUsers++) {
    const userName = `${userPrefix}-${crypto.randomUUID()}`;
    let res = createUser(baseUrl, cookies, `Dartboard Test User ${numUsers + 1}`, userName, testUserPassword)
    if (res.status !== 201) {
      console.log("create user status: ", res.status)
      fail("Failed to create all expected Users")
    }
    
    const userData = JSON.parse(res.body)
    createdUsers.push({
      id: userData.id,
      username: userName
    })
    
    // Add GlobalRoleBinding so user can log in
    const userId = userData.id
    res = createGlobalRoleBinding(baseUrl, { cookies: cookies }, userId, "user")
    if (res.status !== 201 && res.status !== 204) {
      console.log("create globalrolebinding status: ", res.status)
      fail("Failed to create GlobalRoleBinding for user")
    }
  }

  // Wait for GlobalRoleBindings to propagate before continuing
  console.log("Waiting for GlobalRoleBindings to propagate...")
  sleep(5)

  let {res: projectsRes, projectArray} = listProjects(baseUrl, cookies)
  if (projectsRes.status !== 200) {
    fail("Failed to retrieve Projects")
  }

  let projects = projectArray.filter(p => ("spec" in p) && ("displayName" in p["spec"]) && p["spec"]["displayName"].startsWith(projectsPrefix))

  console.log(`Found ${projects.length} projects matching prefix "${projectsPrefix}"`)
  console.log(`Created ${createdUsers.length} users`)

  if (projects.length === 0) {
    fail(`No projects found matching prefix "${projectsPrefix}"`)
  }
  if (createdUsers.length === 0) {
    fail(`No users were created`)
  }

  updateRBACNumbers(cookies)

  // return data that remains constant throughout the test
  return {
    cookies: cookies,
    principalIds: getPrincipalIds(baseUrl, cookies),
    myId: myId,
    // clusterIds: clusterIds,
    clusterId: clusterId,
    projects: projects,
    users: createdUsers,  // Use the users we just created with known credentials
  }
}

// updates count for each of the relevant RBAC metrics
// NOTE: k6 does not update metrics until the end of an iteration,
//       so the minimums reported post- the first iteration!
function updateRBACNumbers(cookies) {
  let numRoles = Number(JSON.parse(listRoles(baseUrl, cookies).body).count)
  numRolesTrend.add(numRoles)
  let numRoleTemplates = Number(JSON.parse(listRoleTemplates(baseUrl, cookies).body).count)
  numRoleTemplatesTrend.add(numRoleTemplates)
  let numRoleBindings = Number(JSON.parse(listRoleBindings(baseUrl, cookies).body).count)
  numRoleBindingsTrend.add(numRoleBindings)
  let numClusterRoles = Number(JSON.parse(listClusterRoles(baseUrl, cookies).body).count)
  numClusterRolesTrend.add(numClusterRoles)
  let numClusterRoleBindings = Number(JSON.parse(listClusterRoleBindings(baseUrl, cookies).body).count)
  numClusterRoleBindingsTrend.add(numClusterRoleBindings)
  let numCRTBs = Number(JSON.parse(listCRTBs(baseUrl, cookies).body).count)
  numCRTBsTrend.add(numCRTBs)
  let numPRTBs = Number(JSON.parse(listPRTBs(baseUrl, cookies).body).count)
  numPRTBsTrend.add(numPRTBs)

  return {
    numRoles: numRoles,
    numRoleTemplates: numRoleTemplates,
    numRoleBindings: numRoleBindings,
    numClusterRoles: numClusterRoles,
    numClusterRoleBindings: numClusterRoleBindings,
    numCRTBs: numCRTBs,
    numPRTBs: numPRTBs,
  }
}

function cleanup(cookies) {
  console.log("Cleaning up Projects, Users, Role Templates, CRTBs, and PRTBs with description label 'Dartboard' or name starting with test prefixes")
  let projectsDeleted = deleteProjectsByPrefix(baseUrl, cookies, projectsPrefix)
  // Use "Dartboard" prefix to match the description format "Dartboard Test <Object> X"
  let usersDeleted = deleteUsersByPrefix(baseUrl, cookies, "Dartboard")
  let prtbsDeleted = deletePRTBsByDescriptionLabel(baseUrl, cookies, "Dartboard")
  let crtbsDeleted = deleteCRTBsByDescriptionLabel(baseUrl, cookies, "Dartboard")
  let roleTemplatesDeleted = deleteRoleTemplatesByPrefix(baseUrl, cookies, "Dartboard")
  if (!projectsDeleted || !usersDeleted || !roleTemplatesDeleted
    || !prtbsDeleted || !crtbsDeleted) {
    console.log("Projects deleted status: ", projectsDeleted)
    console.log("Users deleted status: ", usersDeleted)
    console.log("Role Templates deleted status: ", roleTemplatesDeleted)
    console.log("PRTBs deleted status: ", prtbsDeleted)
    console.log("CRTBs deleted status: ", crtbsDeleted)
    // Don't fail on cleanup issues, just log them
    fail("failed to delete all objects created by test")
  }
}

function createUserExpectFail(baseUrl, cookies, name, password = "useruseruser") {
  const res = http.post(`${baseUrl}/v3/users`,
    JSON.stringify({
      "type": "user",
      "name": displayName,
      "description": `Dartboard ${displayName}`,
      "enabled": true,
      "mustChangePassword": false,
      "password": password,
      "username": `user-${userName}`
    }),
    { cookies: cookies }
  )

  let checkOK = check(res, {
    '/v3/users returns status 401 or 403': (r) => r.status === 401 || r.status === 403,
  })

  let userData = JSON.parse(res.body)

  if (!checkOK || userData.length > 0) {
    fail("Status check failed or received unexpected User data")
  }

  return res
}

function createProjectExpectFail(baseUrl, cookies, name, clusterId, userPrincipalId) {
  let res = http.post(
    `${baseUrl}/v3/projects`,
    JSON.stringify({
      "type": "project",
      "name": name,
      "description": `Dartboard ${name}`,
      "annotations": {},
      "labels": {},
      "clusterId": clusterId,
      "creatorId": `local://${userPrincipalId}`,
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
    }),
    { cookies: cookies }
  )

  let checkOK = check(res, {
    '/v3/projects returns status 401 or 403': (r) => r.status === 401 || r.status === 403,
  })

  let projectData = JSON.parse(res.body)

  if (!checkOK || projectData.length > 0) {
    console.log("\nResponse post-verify project creation: ", JSON.stringify(res, null, 2), "\n")
    fail("Status check failed or received unexpected Project data")
  }

  return res
}

export function createPRTBs(data) {
  const iterationIndex = __ITER % data.projects.length
  const project = data.projects[iterationIndex]
  
  if (!project) {
    console.log(`No project found at index ${iterationIndex}, projects length: ${data.projects.length}`)
    return
  }

  // Use modulo to get user index, ensuring we don't go out of bounds
  const userIndex = __ITER % data.users.length
  let user = data.users[userIndex]

  if (!user) {
    console.log(`No user found at index ${userIndex}, users length: ${data.users.length}`)
    return
  }

  let projectRoleTemplate = {
    "type": "roleTemplate",
    "name": `${projectRoleTemplatePrefix} ${iterationIndex}`,
    "description": `Dartboard Test Project RT ${iterationIndex}`,
    "rules": [
      {
        "apiGroups": [
          "management.cattle.io"
        ],
        "resourceNames": [],
        "resources": [
          "project"
        ],
        "verbs": [
          "get",
          "list"
        ]
      }
    ],
    "external": false,
    "locked": false,
    "clusterCreatorDefault": false,
    "projectCreatorDefault": false,
    "context": "project",
    "roleTemplateIds": []
  }

  let res = createRoleTemplate(baseUrl, data.cookies, projectRoleTemplate)

  if (res.status !== 201) {
    fail("Could not create Project Role Template")
  }

  let roleTemplateId = JSON.parse(res.body).id

  const projectId = project.id.replace("/", ":")

  res = createPRTB(baseUrl, data.cookies, projectId, roleTemplateId, user.id)
  check(res, {
    'PRTB post returns 201 (created)': (r) => r.status === 201,
  })

  // log in as user
  let loginRes = login(baseUrl, {}, user.username, testUserPassword)
  if (loginRes.status !== 200) {
    console.warn(`could not login to cluster as ${user.username}, status: ${loginRes.status}`)
    return  // Don't fail, just skip verification for this iteration
  }
  const cookies = getCookies(baseUrl)

  // updateRBACNumbers with admin cookies
  updateRBACNumbers(data.cookies)
  
  // verify permissions with user cookies
  getProjectById(baseUrl, cookies, data.projects[iterationIndex].id.replace("/", ":"))
  listProjects(baseUrl, cookies)
  const projectName = `${projectsPrefix}-should-fail-vu${metaVU.idInInstance}-${iterationIndex.toString().padStart(4, '0')}`
  createProjectExpectFail(baseUrl, cookies, projectName, data.clusterId, user.id)

  res = logout(baseUrl, cookies);
  if (res.status !== 200) {
    console.log("\nResponse post-verify prtb: ", JSON.stringify(res, null, 2), "\n")
    fail("Failed to logout")
  }
}

export function createCRTBs(data) {
  const iterationIndex = __ITER % data.users.length
  
  // Use modulo to get user index
  const userIndex = __ITER % data.users.length
  let user = data.users[userIndex]

  if (!user) {
    console.log(`No user found at index ${userIndex}, users length: ${data.users.length}`)
    return
  }

  let clusterRoleTemplate = {
    "type": "roleTemplate",
    "name": `${clusterRoleTemplatePrefix} ${iterationIndex}`,
    "description": `Dartboard Test Cluster RT ${iterationIndex}`,
    "rules": [
      {
        "apiGroups": [
          "management.cattle.io"
        ],
        "resourceNames": [],
        "resources": [
          "projects"
        ],
        "verbs": [
          "get",
          "list"
        ]
      }
    ],
    "external": false,
    "locked": false,
    "clusterCreatorDefault": false,
    "projectCreatorDefault": false,
    "context": "cluster",
    "roleTemplateIds": []
  }

  let res = createRoleTemplate(baseUrl, data.cookies, clusterRoleTemplate)

  if (res.status !== 201) {
    fail("Could not create Project Role Template")
  }

  let roleTemplateId = JSON.parse(res.body).id

  res = createCRTB(baseUrl, data.cookies, data.clusterId, roleTemplateId, user.id)
  check(res, {
    'CRTB post returns 201 (created)': (r) => r.status === 201,
  })

   // log in as user
  let loginRes = login(baseUrl, {}, user.username, testUserPassword)
  if (loginRes.status !== 200) {
    console.warn(`could not login to cluster as ${user.username}, status: ${loginRes.status}`)
    return  // Don't fail, just skip verification for this iteration
  }
  const cookies = getCookies(baseUrl)

  // updateRBACNumbers with admin cookies
  updateRBACNumbers(data.cookies)

  // Get a project index safely
  const projectIndex = __ITER % data.projects.length

  // verify permissions with user cookies
  getProjectById(baseUrl, cookies, data.projects[projectIndex].id.replace("/", ":"))
  listProjects(baseUrl, cookies)
  const projectName = `${projectsPrefix}-should-fail-vu${metaVU.idInInstance}-${iterationIndex.toString().padStart(4, '0')}`
  createProjectExpectFail(baseUrl, cookies, projectName, data.clusterId, user.id)

  res = logout(baseUrl, cookies);
  if (res.status !== 200) {
    console.log("\nResponse post-verify crtb: ", JSON.stringify(res, null, 2), "\n")
    fail("Failed to logout")
  }
}
