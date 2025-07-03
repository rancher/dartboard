import { check, fail, sleep } from 'k6'
import http from 'k6/http'

export const userGlobalRoleId = "user"
export const viewRancherMetricsGlobalRoleId = "view-rancher-metrics"
export const viewProjectsClusterRoleId = "projects-view"
export const viewCRTBsClusterRoleId = "clusterroletemplatebindings-view"
export const viewClusterCatalogsClusterRoleId = "clustercatalogs-view"

export function getClusterRoleTemplateBindings(baseUrl, cookies) {
  const res = http.get(`${baseUrl}/v3/clusterroletemplatebindings`,
    { cookies: cookies }
  )
  check(res, {
    'GET /v3/clusterroletemplatebindings returns status 200': (r) => r.status === 201,
  })
  return JSON.parse(res.body)
}

export function createClusterRoleTemplateBinding(baseUrl, cookies, clusterId, roleTemplateId, userPrincipalId) {
  const res = http.post(`${baseUrl}/v3/clusterroletemplatebindings`,
    {
      "type": "clusterroletemplatebinding",
      "clusterId": clusterId,
      "roleTemplateId": roleTemplateId,
      "userPrincipalId": userPrincipalId
    },
    { cookies: cookies }
  )
  check(res, {
    'GET /v3/clusterroletemplatebindings returns status 200': (r) => r.status === 201,
  })
  return res
}

export function deleteClusterRoleTemplateBinding(baseUrl, cookies, crtbId) {
  const res = http.del(`${baseUrl}/v3/clusterroletemplatebindings/${crtbId}`,
    { cookies: cookies }
  )
  check(res, {
    'DELETE /v3/clusterroletemplatebindings returns status 200': (r) => r.status === 200 || r.status === 204,
  })
  return JSON.parse(res.body)
}

export function createGlobalRoleBinding(baseUrl, cookies) {
  const res = http.post(`${baseUrl}/v1/management.cattle.io.globalrolebindings`,
    { cookies: cookies }
  )
  check(res, {
    'POST /v1/management.cattle.io.globalrolebindings returns status 200': (r) => r.status === 200 || r.status === 204,
  })
  return JSON.parse(res.body)
}

export function getGlobalRoleBindings(baseUrl, cookies) {
  const res = http.get(`${baseUrl}/v1/management.cattle.io.globalrolebindings`,
    { cookies: cookies }
  )
  check(res, {
    'GET /v1/management.cattle.io.globalrolebindings returns status 200': (r) => r.status === 200 || r.status === 204,
  })
  return JSON.parse(res.body).data
}

export function deleteGlobalRoleBinding(baseUrl, cookies, id) {
  const res = http.del(`${baseUrl}/v1/management.cattle.io.globalrolebindings/${id}`,
    { cookies: cookies }
  )
  check(res, {
    'DELETE /v1/management.cattle.io.globalrolebindings returns status 200': (r) => r.status === 200 || r.status === 204,
  })
  return JSON.parse(res.body)
}

export function addPermissionToUser(baseUrl, cookies, userId, globalRoleId) {
  const res = http.post(
    `${baseUrl}/v3/globalrolebindings`,
    JSON.stringify({
      "type": "globalRoleBinding",
      "globalRoleId": globalRoleId,
      "userId": userId
    }),
    { cookies: cookies }
  )

  sleep(0.1)
  if (res.status != 201) {
    console.log("CREATE globalRoleBinding failed:\n", JSON.stringify(res, null, 2))
  }
  check(res, {
    'POST /v3/globalrolebindings returns status 201': (r) => r.status === 201 || r.status === 204,
  })
  return res
}

export function giveUserPermission(baseUrl, cookies, userId) {
  const res = addPermissionToUser(baseUrl, cookies, userId, userGlobalRoleId)

  sleep(0.1)
  if (res.status != 201) {
    console.log(`POST '${userGlobalRoleId}' GlobalRoleBinding failed:\n`, JSON.stringify(res, null, 2))
  }
  return JSON.parse(res.body)
}

export function giveViewMetricsPermission(baseUrl, cookies, userId) {
  const res = addPermissionToUser(baseUrl, cookies, userId, viewRancherMetricsGlobalRoleId)

  sleep(0.1)
  if (res.status != 201) {
    console.log(`POST '${viewRancherMetricsGlobalRoleId}' GlobalRoleBinding failed:\n`, JSON.stringify(res, null, 2))
  }
  return JSON.parse(res.body)
}

export function giveViewClusterProjectsPermission(baseUrl, cookies, clusterId, userPrincipalId) {
  const res = createClusterRoleTemplateBinding(baseUrl, cookies, clusterId, viewProjectsClusterRoleId, userPrincipalId)

  sleep(0.1)
  if (res.status != 201) {
    console.log(`POST '${viewProjectsClusterRoleId}' ClusterRoleTemplateBinding failed:\n`, JSON.stringify(res, null, 2))
  }
  return JSON.parse(res.body)
}

export function giveViewCRTBsPermission(baseUrl, cookies, clusterId, userPrincipalId) {
  const res = createClusterRoleTemplateBinding(baseUrl, cookies, clusterId, viewCRTBsClusterRoleId, userPrincipalId)

  sleep(0.1)
  if (res.status != 201) {
    console.log(`POST '${viewCRTBsClusterRoleId}' ClusterRoleTemplateBinding failed:\n`, JSON.stringify(res, null, 2))
  }
  return JSON.parse(res.body)
}

export function giveViewClusterCatalogsPermission(baseUrl, cookies, clusterId, userPrincipalId) {
  const res = createClusterRoleTemplateBinding(baseUrl, cookies, clusterId, viewClusterCatalogsClusterRoleId, userPrincipalId)

  sleep(0.1)
  if (res.status != 201) {
    console.log(`POST '${viewClusterCatalogsClusterRoleId}' ClusterRoleTemplateBinding failed:\n`, JSON.stringify(res, null, 2))
  }
  return JSON.parse(res.body)
}
