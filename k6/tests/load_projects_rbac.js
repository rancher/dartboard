import { check, fail, sleep } from 'k6';
import http from 'k6/http'
import { Trend, Counter } from 'k6/metrics';
import * as k6Util from "../generic/k6_utils.js";
import { getCookies, login, logout } from "../rancher/rancher_utils.js";
import * as userUtil from "../rancher/rancher_users_utils.js"
import * as bindingUtil from "../rancher/rancher_rolebindings_utils.js"
import { vu as metaVU } from 'k6/execution'
import * as projectUtil from "../projects/project_utils.js";
import * as namespaceUtil from "../namespaces/namespace_utils.js";
import * as diagnosticsUtil from "../rancher/rancher_diagnostics.js";
import { retryUntilOneOf } from "../rancher/rancher_utils.js";

const vus = Number(__ENV.VUS || 1)
const projectCount = Number(__ENV.PROJECT_COUNT || 1000)
const namespacesPerProject = Number(__ENV.NAMESPACE_COUNT || 2)
const iterations = Number(__ENV.ITERATIONS || 1)
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD
const token = __ENV.TOKEN
const timingDiffThreshold = 500
const namePrefix = "projects-test"
export const timePolled = new Trend('time_polled', true);

const standardUserName = "user-standard"
const userPassword = __ENV.USER_PASSWORD

// Counters for tracking progress
const projectsCreated = new Counter('projects_created')
const namespacesCreated = new Counter('namespaces_created')
const expectedProjectsPerVU = projectCount
const expectedNamespacesPerVU = expectedProjectsPerVU * namespacesPerProject

// Global variables to track created resources
let createdProjectIds = []
let createdNamespaceIds = []

export const handleSummary = k6Util.customHandleSummary;

export const options = {
  insecureSkipTLSVerify: true,
  scenarios: {
    load: {
      executor: 'per-vu-iterations',
      exec: 'loadProjects',
      vus: vus,
      iterations: iterations,
      maxDuration: '24h',
    }
  },
  thresholds: {
    http_req_failed: ['rate<=0.01'], // http errors should be less than 1%
    http_req_duration: ['p(99)<=500'], // 95% of requests should be below 500ms
    checks: ['rate>0.99'], // the rate of successful checks should be higher than 99%
    [`time_polled{url:'/v1/management.cattle.io.projects'}`]: ['p(99) < 5000', 'avg < 2500'],
    // API response time thresholds - baseline
    [`api_systemimage_duration{phase:'baseline', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_event_duration{phase:'baseline', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_k8sevent_duration{phase:'baseline', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_settings_duration{phase:'baseline', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_clusterrole_duration{phase:'baseline', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_crd_duration{phase:'baseline', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_configmap_duration{phase:'baseline', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_secret_duration{phase:'baseline', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_pod_duration{phase:'baseline', user:'admin'}`]: ['avg<4000', 'p(95)<2000'],
    [`api_project_duration{phase:'baseline', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_namespace_duration{phase:'baseline', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    // API response time thresholds - halfway
    [`api_systemimage_duration{phase:'halfway', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_event_duration{phase:'halfway', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_k8sevent_duration{phase:'halfway', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_settings_duration{phase:'halfway', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_clusterrole_duration{phase:'halfway', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_crd_duration{phase:'halfway', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_configmap_duration{phase:'halfway', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_secret_duration{phase:'halfway', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_pod_duration{phase:'halfway', user:'admin'}`]: ['avg<4000', 'p(95)<2000'],
    [`api_project_duration{phase:'halfway', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_namespace_duration{phase:'halfway', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    // API response time thresholds - final
    [`api_systemimage_duration{phase:'final', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_event_duration{phase:'final', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_k8sevent_duration{phase:'final', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_settings_duration{phase:'final', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_clusterrole_duration{phase:'final', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_crd_duration{phase:'final', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_configmap_duration{phase:'final', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_secret_duration{phase:'final', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_pod_duration{phase:'final', user:'admin'}`]: ['avg<4000', 'p(95)<2000'],
    [`api_project_duration{phase:'final', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_namespace_duration{phase:'final', user:'admin'}`]: ['avg<1000', 'p(95)<2000'],
    // API response time thresholds - baseline
    [`api_systemimage_duration{phase:'baseline', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_event_duration{phase:'baseline', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_k8sevent_duration{phase:'baseline', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_settings_duration{phase:'baseline', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_clusterrole_duration{phase:'baseline', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_crd_duration{phase:'baseline', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_configmap_duration{phase:'baseline', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_secret_duration{phase:'baseline', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_pod_duration{phase:'baseline', user:'user-standard'}`]: ['avg<4000', 'p(95)<2000'],
    [`api_project_duration{phase:'baseline', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_namespace_duration{phase:'baseline', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    // API response time thresholds - halfway
    [`api_systemimage_duration{phase:'halfway', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_event_duration{phase:'halfway', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_k8sevent_duration{phase:'halfway', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_settings_duration{phase:'halfway', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_clusterrole_duration{phase:'halfway', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_crd_duration{phase:'halfway', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_configmap_duration{phase:'halfway', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_secret_duration{phase:'halfway', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_pod_duration{phase:'halfway', user:'user-standard'}`]: ['avg<4000', 'p(95)<2000'],
    [`api_project_duration{phase:'halfway', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_namespace_duration{phase:'halfway', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    // API response time thresholds - final
    [`api_systemimage_duration{phase:'final', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_event_duration{phase:'final', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_k8sevent_duration{phase:'final', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_settings_duration{phase:'final', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_clusterrole_duration{phase:'final', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_crd_duration{phase:'final', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_configmap_duration{phase:'final', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_secret_duration{phase:'final', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_pod_duration{phase:'final', user:'user-standard'}`]: ['avg<4000', 'p(95)<2000'],
    [`api_project_duration{phase:'final', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
    [`api_namespace_duration{phase:'final', user:'user-standard'}`]: ['avg<1000', 'p(95)<2000'],
  },
  setupTimeout: '30m',
  teardownTimeout: '30m',
  // httpDebug: 'full',
}

function cleanup(adminCookies, namePrefix) {
  let deleteAllFailed = false

  // Can't delete rolebindings here because this function is used in setup() and
  // we have no way to track which bindings were created in previous test runs at this point
  // console.log(`Cleaning up created users and bindings`)
  // let delRes = bindingUtil.deleteClusterRoleTemplateBinding(baseUrl, data.userCookies, data.userCRTBId)
  // if (delRes.status !== 200 && delRes.status !== 204) deleteAllFailed = true
  // sleep(0.5)

  // delRes = bindingUtil.deleteGlobalRoleBinding(baseUrl, data.userCookies, data.userGRBId)
  // if (delRes.status !== 200 && delRes.status !== 204) deleteAllFailed = true
  // sleep(0.5)
  // delRes = bindingUtil.deleteGlobalRoleBinding(baseUrl, data.userCookies, data.userMetricsGRBId)
  // if (delRes.status !== 200 && delRes.status !== 204) deleteAllFailed = true
  // sleep(0.5)

  let normanUsers = userUtil.getNormanUsers(baseUrl, { cookies: adminCookies })
  normanUsers.filter(r => r["description"].startsWith("Dartboard Test ")).forEach(r => {
    let delRes = retryUntilOneOf([200, 204], 5, () => userUtil.deleteUser(baseUrl, adminCookies, r.id))
    if (delRes.status !== 200 && delRes.status !== 204) deleteAllFailed = true
    sleep(0.5)
  })

  console.log(`Cleaning up created namespaces and projects with prefix: '${namePrefix}'`)
  let { __, namespaceArray } = namespaceUtil.getNamespacesMatchingName(baseUrl, adminCookies, namePrefix)
  console.log(`Found ${namespaceArray.length} namespaces to clean up`)
  namespaceArray.forEach(r => {
    let delRes = retryUntilOneOf([200, 204], 5, () => namespaceUtil.deleteNamespace(baseUrl, adminCookies, r["id"]))
    if (delRes.status !== 200 && delRes.status !== 204) deleteAllFailed = true
    sleep(0.1)
  })

  sleep(2)

  let { _, projectArray } = projectUtil.getNormanProjectsMatchingName(baseUrl, adminCookies, namePrefix)
  console.log(`Found ${projectArray.length} projects to clean up`)
  projectArray.forEach(r => {
    let delRes = retryUntilOneOf([200, 204], 5, () => projectUtil.deleteNormanProject(baseUrl, adminCookies, r["id"]))
    if (delRes.status !== 200 && delRes.status !== 204) deleteAllFailed = true
    sleep(0.1)
  })
  return deleteAllFailed
}

export function setup() {
  console.log(`Starting load test with ${projectCount} projects and ${namespacesPerProject} namespaces per project`)
  var cookies = {}

  let adminLoginRes = login(baseUrl, {}, username, password)

  if (adminLoginRes.status !== 200) {
    fail(`could not login to cluster as admin`)
  }
  cookies = getCookies(baseUrl)
  console.log("ADMIN COOKIES: ", cookies)

  sleep(2)

  // delete leftovers, if any
  let deleteAllFailed = cleanup(cookies, namePrefix)
  if (deleteAllFailed) fail("Failed to delete all existing test projects during setup!")

  console.log("Creating standard user with view-rancher-metrics permissions")
  let userData = userUtil.createUser(baseUrl, { cookies: cookies }, userPassword, "standard")
  const userPrincipalId = userData.principalIds[0]
  const userId = userData.id

  let userGlobalRoleBindingData = bindingUtil.giveUserPermission(baseUrl, cookies, userId)
  let viewMetricsGlobalRoleBindingData = bindingUtil.giveViewMetricsPermission(baseUrl, cookies, userId)
  bindingUtil.giveViewClusterProjectsPermission(baseUrl, cookies, "local", userPrincipalId)
  bindingUtil.giveViewClusterCatalogsPermission(baseUrl, cookies, "local", userPrincipalId)
  bindingUtil.giveViewCRTBsPermission(baseUrl, cookies, "local", userPrincipalId)
  let crtbData = JSON.parse(bindingUtil.createClusterRoleTemplateBinding(baseUrl, cookies, "local", "cluster-member", userPrincipalId).body)
  const userGRBId = userGlobalRoleBindingData.id
  const userMetricsGRBId = viewMetricsGlobalRoleBindingData.id
  const userCRTBId = crtbData.id

  console.log("Logging in with standard user")
  let userLoginRes = login(baseUrl, {}, standardUserName, "useruseruser")
  if (userLoginRes.status !== 200) {
    fail(`could not login to cluster as standard user`)
  }
  // See https://grafana.com/docs/k6/latest/using-k6/cookies/ for cookie-handling logic
  var userCookies = {}
  userCookies = { "R_SESS": [userLoginRes.cookies.R_SESS[0].value] }
  console.log("ADMIN COOKIES: ", cookies)
  console.log("USER COOKIES: ", userCookies)

  sleep(2)

  // return data that remains constant throughout the test
  return {
    adminCookies: cookies,
    adminPrincipalId: userUtil.getCurrentUserPrincipalId(baseUrl, cookies),
    userCookies: userCookies,
    userPrincipalId: userPrincipalId,
    userId: userId,
    userGRBId: userGRBId,
    userMetricsGRBId: userMetricsGRBId,
    userCRTBId: userCRTBId
  }
}

// export function teardown(data) {
//   cleanup(data, namePrefix)
//   logout(baseUrl, data.userCookies)
//   logout(baseUrl, data.adminCookies)
// }

function collectResourceCounts(cookies, phase) {
  console.log(`Collecting resource counts for phase: ${phase}`)

  let resourceCountsRaw = diagnosticsUtil.getLocalClusterResourceCounts(baseUrl, cookies)
  let resourceCounts = diagnosticsUtil.processResourceCounts(resourceCountsRaw)

  // Record key resource counts as metrics
  if (resourceCounts.projects && resourceCounts.projects.totalCount !== undefined) {
    diagnosticsUtil.totalProjectsGauge.add(resourceCounts.projects.totalCount, { phase: phase })
    console.log(`${phase} - Total Projects: ${resourceCounts.projects.totalCount}`)
  }

  if (resourceCounts.namespaces && resourceCounts.namespaces.totalCount !== undefined) {
    diagnosticsUtil.totalNamespacesGauge.add(resourceCounts.namespaces.totalCount, { phase: phase })
    console.log(`${phase} - Total Namespaces: ${resourceCounts.namespaces.totalCount}`)
  }

  if (resourceCounts.pods && resourceCounts.pods.totalCount !== undefined) {
    diagnosticsUtil.totalPodsGauge.add(resourceCounts.pods.totalCount, { phase: phase })
    console.log(`${phase} - Total Pods: ${resourceCounts.pods.totalCount}`)
  }

  if (resourceCounts.secrets && resourceCounts.secrets.totalCount !== undefined) {
    diagnosticsUtil.totalSecretsGauge.add(resourceCounts.secrets.totalCount, { phase: phase })
    console.log(`${phase} - Total Secrets: ${resourceCounts.secrets.totalCount}`)
  }

  if (resourceCounts.configmaps && resourceCounts.configmaps.totalCount !== undefined) {
    diagnosticsUtil.totalConfigMapsGauge.add(resourceCounts.configmaps.totalCount, { phase: phase })
    console.log(`${phase} - Total ConfigMaps: ${resourceCounts.configmaps.totalCount}`)
  }

  if (resourceCounts.serviceaccounts && resourceCounts.serviceaccounts.totalCount !== undefined) {
    diagnosticsUtil.totalServiceAccountsGauge.add(resourceCounts.serviceaccounts.totalCount, { phase: phase })
    console.log(`${phase} - Total ServiceAccounts: ${resourceCounts.serviceaccounts.totalCount}`)
  }

  if (resourceCounts.clusterroles && resourceCounts.clusterroles.totalCount !== undefined) {
    diagnosticsUtil.totalClusterRolesGauge.add(resourceCounts.clusterroles.totalCount, { phase: phase })
    console.log(`${phase} - Total ClusterRoles: ${resourceCounts.clusterroles.totalCount}`)
  }

  if (resourceCounts.customresourcedefinitions && resourceCounts.customresourcedefinitions.totalCount !== undefined) {
    diagnosticsUtil.totalCRDsGauge.add(resourceCounts.customresourcedefinitions.totalCount, { phase: phase })
    console.log(`${phase} - Total CRDs: ${resourceCounts.customresourcedefinitions.totalCount}`)
  }

  return resourceCounts
}

function collectAPITimings(cookies, phase, user) {
  console.log(`Collecting API response timings for phase: ${phase} user: ${user}`)
  let timings = diagnosticsUtil.processResourceTimings(baseUrl, cookies)

  // Record each API timing with phase and user tags
  const timingsTags = { phase: phase, user: user }

  if (timings["management.cattle.io.rkek8ssystemimage"]) {
    diagnosticsUtil.systemImageAPITime.add(timings["management.cattle.io.rkek8ssystemimage"], timingsTags)
  }

  if (timings["event"]) {
    diagnosticsUtil.eventAPITime.add(timings["event"], timingsTags)
  }

  if (timings["events.k8s.io.event"]) {
    diagnosticsUtil.k8sEventAPITime.add(timings["events.k8s.io.event"], timingsTags)
  }

  if (timings["management.cattle.io.setting"]) {
    diagnosticsUtil.settingsAPITime.add(timings["management.cattle.io.setting"], timingsTags)
  }

  if (timings["rbac.authorization.k8s.io.clusterrole"]) {
    diagnosticsUtil.clusterRoleAPITime.add(timings["rbac.authorization.k8s.io.clusterrole"], timingsTags)
  }

  if (timings["apiextensions.k8s.io.customresourcedefinition"]) {
    diagnosticsUtil.crdAPITime.add(timings["apiextensions.k8s.io.customresourcedefinition"], timingsTags)
  }

  if (timings["rbac.authorization.k8s.io.role"]) {
    diagnosticsUtil.roleAPITime.add(timings["rbac.authorization.k8s.io.role"], timingsTags)
  }

  if (timings["rbac.authorization.k8s.io.rolebinding"]) {
    diagnosticsUtil.roleBindingAPITime.add(timings["rbac.authorization.k8s.io.rolebinding"], timingsTags)
  }

  if (timings["rbac.authorization.k8s.io.clusterrolebinding"]) {
    diagnosticsUtil.clusterRoleBindingAPITime.add(timings["rbac.authorization.k8s.io.clusterrolebinding"], timingsTags)
  }

  if (timings["management.cattle.io.globalrolebinding"]) {
    diagnosticsUtil.globalRoleBindingAPITime.add(timings["management.cattle.io.globalrolebinding"], timingsTags)
  }

  if (timings["management.cattle.io.rkeaddon"]) {
    diagnosticsUtil.rkeAddonAPITime.add(timings["management.cattle.io.rkeaddon"], timingsTags)
  }

  if (timings["configmap"]) {
    diagnosticsUtil.configMapAPITime.add(timings["configmap"], timingsTags)
  }

  if (timings["serviceaccount"]) {
    diagnosticsUtil.serviceAccountAPITime.add(timings["serviceaccount"], timingsTags)
  }

  if (timings["secret"]) {
    diagnosticsUtil.secretAPITime.add(timings["secret"], timingsTags)
  }

  if (timings["pod"]) {
    diagnosticsUtil.podAPITime.add(timings["pod"], timingsTags)
  }

  if (timings["management.cattle.io.rkek8sserviceoption"]) {
    diagnosticsUtil.rkeServiceOptionAPITime.add(timings["management.cattle.io.rkek8sserviceoption"], timingsTags)
  }

  if (timings["apiregistration.k8s.io.apiservice"]) {
    diagnosticsUtil.apiServiceAPITime.add(timings["apiregistration.k8s.io.apiservice"], timingsTags)
  }

  if (timings["management.cattle.io.roletemplate"]) {
    diagnosticsUtil.roleTemplateAPITime.add(timings["management.cattle.io.roletemplate"], timingsTags)
  }

  if (timings["management.cattle.io.project"]) {
    diagnosticsUtil.projectAPITime.add(timings["management.cattle.io.project"], timingsTags)
  }

  if (timings["namespace"]) {
    diagnosticsUtil.namespaceAPITime.add(timings["namespace"], timingsTags)
  }

  let slowestApis = Object.entries(timings)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 5)

  console.log(`${phase} - Top 5 slowest APIs:`)
  slowestApis.forEach(([api, time], index) => {
    console.log(`  ${index + 1}. ${api}: ${time.toFixed(2)}ms`)
  })

  return timings
}

function getProjectWithRetry(baseUrl, cookies, projectId, maxRetries = 5) {
  for (let retry = 0; retry < maxRetries; retry++) {
    let { res, project } = projectUtil.getProject(baseUrl, cookies, projectId)
    if (res.status == 200 && project && project.status.conditions) {
      return project
    }
    console.warn(`GET Project attempt #${retry + 1} failed with status ${res.status}`)
    sleep(1)
  }
  return {}
}

function createNormanProjectWithRetry(baseUrl, projectBody, cookies, maxRetries = 5) {
  for (let retry = 0; retry < maxRetries; retry++) {
    const res = projectUtil.createNormanProject(baseUrl, projectBody, cookies)
    if (res.status === 201) {
      return res
    }
    console.warn(`POST Project attempt #${retry + 1} failed with status ${res.status}`)
    sleep(1)
  }
  return {}
}

function createProjectsAndNamespaces(data, startIndex, endIndex) {
  console.log(`Creating projects from ${startIndex} to ${endIndex}`)
  console.debug("CURRENT USER PRINCIPAL ID: ", data.adminPrincipalId)

  for (let i = startIndex; i < endIndex; i++) {
    const projectName = `${namePrefix}-${metaVU.idInInstance}-${i.toString().padStart(4, '0')}`
    const projectBody = JSON.stringify({
      type: "project",
      name: projectName,
      description: `Load test project ${i + 1}`,
      clusterId: "local",
      creatorId: data.adminPrincipalId,
      labels: {},
      annotations: {},
      resourceQuota: {},
      containerDefaultResourceLimit: {},
      namespaceDefaultResourceQuota: {}
    })

    console.log(`Creating project ${i + 1}/${projectCount}: ${projectName}`)
    const projectRes = createNormanProjectWithRetry(baseUrl, projectBody, data.adminCookies, 5)

    if (projectRes.status === 201) {
      const projectData = JSON.parse(projectRes.body)
      createdProjectIds.push(projectData.id)
      projectsCreated.add(1)
      let projectId = projectData.id.replace(":", "/")
      let steveProject = getProjectWithRetry(baseUrl, data.adminCookies, projectId, 5)

      // Create namespaces for this project
      for (let j = 0; j < namespacesPerProject; j++) {
        const namespaceName = `${namePrefix}-${metaVU.idInInstance}ns-${i.toString().padStart(4, '0')}-${j + 1}`
        const namespaceBody = JSON.stringify({
          "type": "namespace",
          "disableOpenApiValidation": false,
          "metadata": {
            "name": namespaceName,
            "labels": {
              "field.cattle.io/projectId": steveProject.metadata.name
            },
            "annotations": {
              "field.cattle.io/containerDefaultResourceLimit": "{}",
              "field.cattle.io/description": `Load test namespace ${i + j + 1}`,
              "field.cattle.io/projectId": projectData.id
            },
          },
        })

        console.log(`  Creating namespace ${j + 1}/${namespacesPerProject}: ${namespaceName}`)
        const namespaceRes = namespaceUtil.createNamespace(baseUrl, namespaceBody, data.adminCookies)

        if (namespaceRes.status === 201) {
          const namespaceData = JSON.parse(namespaceRes.body)
          createdNamespaceIds.push(namespaceData.id)
          namespacesCreated.add(1)
        } else {
          console.error(`Failed to create namespace ${namespaceName}: ${namespaceRes.status}`)
        }

        sleep(0.5)
      }
    } else {
      console.error(`Failed to create project ${projectName}: ${projectRes.status}`)
    }

    if ((i + 1) % 50 === 0) {
      console.log(`Progress: ${i + 1}/${endIndex} projects created`)
    }

    sleep(0.5)
  }
}

export function loadProjects(data) {
  console.log("=== Starting Load Test Execution ===")

  // Step 1: Baseline measurement (before creating any projects)
  console.log('Step 1: Collecting initial diagnostics...')
  let resourceCounts = collectResourceCounts(data.adminCookies, 'baseline')
  diagnosticsUtil.processMetricsFromCountData(resourceCounts)
  let adminTimings = collectAPITimings(data.adminCookies, 'baseline', 'admin')
  sleep(5)
  let userTimings = collectAPITimings(data.userCookies, 'baseline', standardUserName)
  Object.keys(adminTimings).forEach(key => {
    if (userTimings.hasOwnProperty(key)) {
      let adminValue = adminTimings[key]
      let userValue = userTimings[key]
      let threshold = userValue + timingDiffThreshold

      let checkName = `API Response Timing comparison for ${key}`
      let passed = adminValue <= threshold

      check(null, {
        [checkName]: () => passed,
      })
      if (!passed) {
        console.warn(`Timing Diff surpassed threshold for ${key}:`)
        console.warn(`   Admin: ${adminValue}ms, User: ${userValue}ms`)
        console.warn(`   Diff: +${adminValue - userValue}ms (threshold: ${timingDiffThreshold}ms)`)
      } else {
        console.log(`${key}: ${adminValue}ms (User: ${userValue}ms, Diff: ${adminValue - userValue > 0 ? '+' : ''}${adminValue - userValue}ms)`)
      }
    } else {
      console.info(`New metric found: ${key} = ${adminTimings[key]}ms (no equivalent metric for the given non-admin user is available)`)
    }
  })

  // Step 2: Create first half of projects
  const halfwayPoint = Math.floor(projectCount / 2)
  console.log(`Step 2: Creating first half of projects (0 to ${halfwayPoint})...`)
  createProjectsAndNamespaces(data, 0, halfwayPoint)

  // Step 3: Halfway measurement
  console.log('Step 3: Collecting diagnostics at halfway point...')
  resourceCounts = collectResourceCounts(data.adminCookies, 'halfway')
  diagnosticsUtil.processMetricsFromCountData(resourceCounts)
  collectAPITimings(data.adminCookies, 'halfway', 'admin')
  sleep(5)
  collectAPITimings(data.userCookies, 'halfway', standardUserName)

  // Step 4: Create remaining projects
  console.log(`Step 4: Creating remaining projects (${halfwayPoint} to ${projectCount})...`)
  createProjectsAndNamespaces(data, halfwayPoint, projectCount)

  // Step 5: Final measurement
  console.log('Step 5: Collecting final diagnostics...')
  resourceCounts = collectResourceCounts(data.adminCookies, 'final')
  diagnosticsUtil.processMetricsFromCountData(resourceCounts)
  collectAPITimings(data.adminCookies, 'final', 'admin')
  sleep(5)
  collectAPITimings(data.userCookies, 'final', standardUserName)

  let projectCriteria = []
  let numProjects = createdProjectIds.length
  let numNamespaces = createdNamespaceIds.length
  projectCriteria[`expected # of projects (${expectedProjectsPerVU}) were created`] = (p) => p === expectedProjectsPerVU
  check(numProjects, projectCriteria)
  let namespaceCriteria = []
  namespaceCriteria[`expected # of namespaces (${expectedNamespacesPerVU}) were created`] = (n) => n === expectedNamespacesPerVU
  check(numNamespaces, namespaceCriteria)

  console.log("=== Load Test Execution Complete ===")
  console.log(`Total projects created: ${numProjects}`)
  console.log(`Total namespaces created: ${numNamespaces}`)
  console.log(`  Expected total: ${expectedProjectsPerVU} projects, ${expectedNamespacesPerVU} namespaces`)
  console.log("=== Final Resource Counts ===")
  console.log(JSON.stringify(resourceCounts, null, 2))
}
