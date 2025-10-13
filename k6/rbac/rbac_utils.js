import { check, fail } from 'k6';
import http from 'k6/http';

/*
  Usernames are prefixed with "user", password defaults to "useruseruser" if not set
*/
export function createUser(baseUrl, cookies, displayName, userName, password = "useruseruser") {
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

  check(res, {
    '/v3/users returns status 201': (r) => r.status === 201,
  })

  return res
}

export function getUserById(baseUrl, cookies, userId) {
  let res = http.get(`${baseUrl}/v3/users?id=${userId}`, { cookies: cookies })

  let checkOK = check(res, {
    '/v1/management.cattle.io.users returns status 200': (r) => r.status === 200 || r.status === 204,
  })

  let userData = JSON.parse(res.body)["data"]

  if (!checkOK || userData.length !== 1) {
    fail("Status check failed or did not receive User data")
  }

  return res
}

export function listUsers(baseUrl, cookies) {
  let res = http.get(`${baseUrl}/v3/users`, { cookies: cookies })

  let checkOK = check(res, {
    '/v1/management.cattle.io.users returns status 200': (r) => r.status === 200 || r.status === 204,
  })

  let usersData = JSON.parse(res.body)["data"]

  if (!checkOK || usersData === undefined || usersData.length == 0) {
    fail("Status check failed or did not receive list of Users data")
  }

  return res
}

export function listRoleTemplates(baseUrl, cookies) {
  let res = http.get(`${baseUrl}/v1/management.cattle.io.roletemplates`, { cookies: cookies })

  let checkOK = check(res, {
    '/v1/management.cattle.io.roletemplates returns status 200': (r) => r.status === 200 || r.status === 204,
  })

  let templatesData = JSON.parse(res.body)["data"]

  if (!checkOK || templatesData === undefined || templatesData.length == 0) {
    fail("Status check failed or did not receive list of RoleTemplates data")
  }

  return res
}

export function listRoles(baseUrl, cookies) {
  let res = http.get(`${baseUrl}/v1/rbac.authorization.k8s.io.roles`, { cookies: cookies })

  let checkOK = check(res, {
    '/v1/rbac.authorization.k8s.io.roles returns status 200': (r) => r.status === 200 || r.status === 204,
  })

  let rolesData = JSON.parse(res.body)["data"]

  if (!checkOK || rolesData === undefined || rolesData.length == 0) {
    fail("Status check failed or did not receive expected Roles data")
  }

  return res
}

export function listClusterRoles(baseUrl, cookies) {
  let res = http.get(`${baseUrl}/v1/rbac.authorization.k8s.io.clusterroles`, { cookies: cookies })

  let checkOK = check(res, {
    '/v1/rbac.authorization.k8s.io.clusterroles returns status 200': (r) => r.status === 200 || r.status === 204,
  })

  let rolesData = JSON.parse(res.body)["data"]

  if (!checkOK || rolesData === undefined || rolesData.length == 0) {
    fail("Status check failed or did not receive list of ClusterRoles data")
  }

  return res
}

export function listRoleBindings(baseUrl, cookies) {
  let res = http.get(`${baseUrl}/v1/rbac.authorization.k8s.io.rolebindings`, { cookies: cookies })

  let checkOK = check(res, {
    '/v1/rbac.authorization.k8s.io.rolebindings returns status 200': (r) => r.status === 200 || r.status === 204,
  })

  let bindingsData = JSON.parse(res.body)["data"]

  if (!checkOK || bindingsData === undefined || bindingsData.length == 0) {
    fail("Status check failed or did not receive list of RoleBindings data")
  }

  return res
}

export function listClusterRoleBindings(baseUrl, cookies) {
  let res = http.get(`${baseUrl}/v1/rbac.authorization.k8s.io.clusterrolebindings`, { cookies: cookies })

  let checkOK = check(res, {
    '/v1/rbac.authorization.k8s.io.clusterrolebindings returns status 200': (r) => r.status === 200 || r.status === 204,
  })

  let bindingsData = JSON.parse(res.body)["data"]

  if (!checkOK || bindingsData === undefined || bindingsData.length == 0) {
    fail("Status check failed or did not receive list of ClusterRoleBindings data")
  }

  return res
}

export function listCRTBs(baseUrl, cookies) {
  let res = http.get(`${baseUrl}/v1/management.cattle.io.clusterroletemplatebindings`, { cookies: cookies })

  let checkOK = check(res, {
    '/v1/management.cattle.io.clusterroletemplatebindings returns status 200': (r) => r.status === 200 || r.status === 204,
  })

  let templatesData = JSON.parse(res.body)["data"]

  if (!checkOK || templatesData === undefined || templatesData.length == 0) {
    fail("Status check failed or did not receive list of ClusterRoleTemplateBindings data")
  }

  return res
}

export function listPRTBs(baseUrl, cookies) {
  let res = http.get(`${baseUrl}/v1/management.cattle.io.projectroletemplatebindings`, { cookies: cookies })

  let checkOK = check(res, {
    '/v1/management.cattle.io.projectroletemplatebindings returns status 200': (r) => r.status === 200 || r.status === 204,
  })

  let templatesData = JSON.parse(res.body)["data"]

  if (!checkOK || templatesData === undefined || templatesData.length == 0) {
    fail("Status check failed or did not receive list of ProjectRoleTemplateBindings data")
  }

  return res
}

export function getGlobalRoles(baseUrl, cookies) {
  let res = http.get(`${baseUrl}/v1/management.cattle.io.globalroles`, { cookies: cookies })
  check(res, {
    '/v1/management.cattle.io.globalroles returns status 200': (r) => r.status === 200 || r.status === 204,
  })
  return res
}

// Create a new RoleTemplate for the given context
/*
  Example:
  baseUrl = "myUrl"
  cookies = "my cookies"
  template = {
      "name": "child-role1",
      "apiGroups": [
        "management.cattle.io",
        "rbac.authorization.k8s.io"
      ],
      "resources": [
        "projects",
        "clusterroles"
      ],
      "verbs": [
      ["get", "list"],
      ["get", "list", "create"]
      ],
      "locked": false,
      "clusterCreatorDefault": false,
      "projectCreatorDefault": false,
      "context": "cluster" || "global" || "project",
      "roleTemplateIds": ["inheritedRoleTemplateIds"]
  }
  const res = createRoleTemplate(baseUrl, cookies, template)
*/
const defaultRoleTemplate = {
  "type": "roleTemplate",
  "name": "defaultDartboardRoleTemplate",
  "description": "Dartboard",
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
  "locked": false,
  "clusterCreatorDefault": false,
  "projectCreatorDefault": false,
  "context": "cluster",
  "roleTemplateIds": []
}
const inheritedRoleTemplate = {
  "name": "defaultInheritedRoleTemplate",
  "description": "Dartboard",
  "locked": false,
  "clusterCreatorDefault": false,
  "projectCreatorDefault": false,
  "context": "cluster",
  "roleTemplateIds": ["cluster-member"]
}
export function createRoleTemplate(baseUrl, cookies, template) {
  const res = http.post(`${baseUrl}/v3/roletemplates`,
    JSON.stringify(template),
    {
      headers: {
        accept: 'application/json',
      },
      cookies: cookies,
    }
  )
  check(res, {
    '/v3/roletemplate returns status 201': (r) => r.status === 201,
  })

  return res
}

export function createGlobalRoleBinding(baseUrl, params, userId, roles = ["user"]) {
  const res = http.post(
    `${baseUrl}/v3/globalrolebindings`,
    JSON.stringify({
      "type": "globalRoleBinding",
      "globalRoleId": roles,
      "userId": userId
    }),
    params
  )

  check(res, {
    '/v3/globalrolebindings returns status 201': (r) => r.status === 201 || r.status === 204,
  })

  return res
}


export function createPRTB(baseUrl, cookies, projectId, roleTemplateId, userId) {
  const res = http.post(
    `${baseUrl}/v3/projectroletemplatebindings`,
    JSON.stringify({
      "type": "projectroletemplatebinding",
      "labels": {
        "description": "Dartboard"
      },
      "roleTemplateId": roleTemplateId,
      "userPrincipalId": `local://${userId}`,
      "projectId": projectId.replace("/", ":")
    }),
    {
      headers: {
        accept: 'application/json',
        'content-type': 'application/json',
      },
      cookies: cookies
    }
  )

  check(res, {
    '/v3/projectroletemplatebindings returns status 201 or 404': (r) => r.status === 201 || r.status === 404,
  })

  return res
}

export function createCRTB(baseUrl, cookies, clusterId, roleTemplateId, userId) {
  const res = http.post(`${baseUrl}/v3/clusterroletemplatebindings`,
    JSON.stringify({
      "type": "clusterroletemplatebinding",
      "labels": {
        "description": "Dartboard"
      },
      "clusterId": clusterId,
      "roleTemplateId": roleTemplateId,
      "userPrincipalId": `local://${userId}`
    }),
    {
      headers: {
        accept: 'application/json',
        'content-type': 'application/json',
      },
      cookies: cookies
    }
  )

  check(res, {
    'POST v3/clusterroletemplatebindings returns status 201': (r) => r.status === 201,
  })

  return res
}

export function deleteUsersByPrefix(baseUrl, cookies, prefix = "Dartboard ") {
  let deletedAll = true

  let res = listUsers(baseUrl, cookies)
  if (res.status !== 200) {
    console.log("list users status: ", res.status)
    return false
  }

  JSON.parse(res.body)["data"].filter(r => ("description" in r) && r["description"].startsWith(prefix)).forEach(r => {
    res = http.del(`${baseUrl}/v3/users/${r["id"]}`, { cookies: cookies })

    if (res.status !== 200 && res.status !== 204) {
      console.log("delete user status: ", res.status)
      deletedAll = false
    }

    check(res, {
      'DELETE /v3/users returns status 200': (r) => r.status === 200 || r.status === 204,
    })
  })

  return deletedAll
}

export function deleteGlobalRolesByPrefix(baseUrl, cookies, prefix = "Dartboard ") {
  let deletedAll = true

  let res = getGlobalRoles(baseUrl, cookies)
  if (res.status !== 200) {
    return false
  }

  JSON.parse(res.body)["data"].filter(r => ("description" in r) && r["description"].startsWith(prefix)).forEach(r => {
    res = http.del(`${baseUrl}/v3/globalRoles/${r["id"]}`, { cookies: cookies })

    if (res.status !== 200) {
      deletedAll = false
    }

    check(res, {
      'DELETE /v3/globalRoles returns status 200': (r) => r.status === 200 || r.status === 204,
    })
  })

  return deletedAll
}

export function deleteRoleTemplatesByPrefix(baseUrl, cookies, prefix = "Dartboard ") {
  let deletedAll = true

  let res = listRoleTemplates(baseUrl, cookies)
  if (res.status !== 200) {
    console.log("list roletemplates status: ", res.status)
    return false
  }

  JSON.parse(res.body)["data"].filter(r => ("description" in r) && r["description"].startsWith(prefix)).forEach(r => {
    res = http.del(`${baseUrl}/v3/roletemplates/${r["id"].replace("/", ":")}`, { cookies: cookies })


    if (res.status !== 200) {
      console.log("delete roletemplates status: ", res.status)
      deletedAll = false
    }

    check(res, {
      'DELETE /v3/roletemplates returns status 200': (r) => r.status === 200,
    })
  })

  return deletedAll
}

export function deleteCRTBsByDescriptionLabel(baseUrl, cookies, label = { "description": "Dartboard" }) {
  let deletedAll = true

  let res = listCRTBs(baseUrl, cookies)
  if (res.status !== 200) {
    console.log("list clusterroletemplatebindings status: ", res.status)
    return false
  }

  JSON.parse(res.body)["data"].filter(r => ("labels" in r) && ("description" in r["labels"]) && r["labels"].description == label["description"]).forEach(r => {
    res = http.del(`${baseUrl}/v3/clusterroletemplatebindings/${r["id"]}`, { cookies: cookies })


    if (res.status !== 200) {
      console.log("delete clusterroletemplatebindings status: ", res.status)
      deletedAll = false
    }

    check(res, {
      'DELETE /v3/clusterroletemplatebindings returns status 200': (r) => r.status === 200,
    })
  })

  return deletedAll
}

export function deletePRTBsByDescriptionLabel(baseUrl, cookies, label = { "description": "Dartboard" }) {
  let deletedAll = true

  let res = listPRTBs(baseUrl, cookies)
  if (res.status !== 200) {
    console.log("list projectroletemplatebindings status: ", res.status)
    return false
  }

  JSON.parse(res.body)["data"].filter(r => ("labels" in r) && ("description" in r["labels"]) && r["labels"].description == label["description"]).forEach(r => {
    res = http.del(`${baseUrl}/v3/projectroletemplatebindings/${r["id"]}`, { cookies: cookies })


    if (res.status !== 200) {
      console.log("delete projectroletemplatebindings status: ", res.status)
      deletedAll = false
    }

    check(res, {
      'DELETE /v3/projectroletemplatebindings returns status 200': (r) => r.status === 200,
    })
  })

  return deletedAll
}
