import { check, fail, sleep } from 'k6'
import { Trend, Gauge } from 'k6/metrics';
import http from 'k6/http'


// Resource count tracking metrics (using Gauges since they represent current state)
export const totalProjectsGauge = new Gauge('cluster_projects_total')
export const totalNamespacesGauge = new Gauge('cluster_namespaces_total')
export const totalPodsGauge = new Gauge('cluster_pods_total')
export const totalSecretsGauge = new Gauge('cluster_secrets_total')
export const totalConfigMapsGauge = new Gauge('cluster_configmaps_total')
export const totalServiceAccountsGauge = new Gauge('cluster_serviceaccounts_total')
export const totalRolesGauge = new Gauge('roles_total')
export const totalRoleBindingsGauge = new Gauge('cluster_rolebindings_total')
export const totalClusterRolesGauge = new Gauge('cluster_clusterroles_total')
export const totalClusterRoleBindingsGauge = new Gauge('cluster_clusterrolebindings_total')
export const totalGlobalRoleBindingsGauge = new Gauge('cluster_globalrolebindings_total')
export const totalCRDsGauge = new Gauge('cluster_crds_total')

// API response time tracking metrics
export const systemImageAPITime = new Trend('api_systemimage_duration')
export const eventAPITime = new Trend('api_event_duration')
export const k8sEventAPITime = new Trend('api_k8sevent_duration')
export const settingsAPITime = new Trend('api_settings_duration')
export const clusterRoleAPITime = new Trend('api_clusterrole_duration')
export const crdAPITime = new Trend('api_crd_duration')
export const roleAPITime = new Trend('api_role_duration')
export const roleBindingAPITime = new Trend('api_rolebinding_duration')
export const clusterRoleBindingAPITime = new Trend('api_clusterrolebinding_duration')
export const globalRoleBindingAPITime = new Trend('api_globalrolebinding_duration')
export const rkeAddonAPITime = new Trend('api_rkeaddon_duration')
export const configMapAPITime = new Trend('api_configmap_duration')
export const serviceAccountAPITime = new Trend('api_serviceaccount_duration')
export const secretAPITime = new Trend('api_secret_duration')
export const podAPITime = new Trend('api_pod_duration')
export const rkeServiceOptionAPITime = new Trend('api_rkeserviceoption_duration')
export const apiServiceAPITime = new Trend('api_apiservice_duration')
export const roleTemplateAPITime = new Trend('api_roletemplate_duration')
export const projectAPITime = new Trend('api_project_duration')
export const namespaceAPITime = new Trend('api_namespace_duration')

export const resourceCount = new Gauge('resource_count')
export const namespaceResourceDensity = new Trend('namespace_resource_density');
export const resourceDistribution = new Trend('resource_distribution')

export const timingTag = { timing: "yes" }
export const clusterScopeTag = { scope: "cluster" }
export const namespaceScopeTag = { scope: "namespace" }
export const localClusterTag = { cluster: "local" }

export function processMetricsFromCountData(data) {
  Object.keys(data).forEach(resourceType => {
    const resourceData = data[resourceType];
    if (resourceData.totalCount !== undefined) {
      resourceCount.add(resourceData.totalCount, { resource_type: resourceType, ...clusterScopeTag, ...localClusterTag });

      // If it has namespace-specific data, create namespace-specific metrics
      if (resourceData.namespaces) {
        Object.entries(resourceData.namespaces).forEach(([namespace, count]) => {
          resourceCount.add(count, { resource_type: resourceType, namespace: namespace, ...namespaceScopeTag })
          namespaceResourceDensity.add(count, { namespace: namespace, resource_type: resourceType })

          // Calculate namespace distribution for this resource type
          const namespaceCounts = Object.values(resourceData.namespaces)
          const avgPerNamespace = namespaceCounts.reduce((a, b) => a + b, 0) / namespaceCounts.length
          const maxPerNamespace = Math.max(...namespaceCounts)
          const minPerNamespace = Math.min(...namespaceCounts)

          resourceDistribution.add(avgPerNamespace, { resource_type: resourceType, metric: 'avg_per_namespace' })
          resourceDistribution.add(maxPerNamespace, { resource_type: resourceType, metric: 'max_per_namespace' })
          resourceDistribution.add(minPerNamespace, { resource_type: resourceType, metric: 'min_per_namespace' })

        });
      }
    }
  });
}

export function getLocalClusterResourceCounts(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/counts`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
  })
  check(response, {
    'k8s/clusters/local/v1/counts can be queried': (r) => r.status === 200,
  })
  return JSON.parse(response.body).data[0].counts
}

export function processResourceCounts(resourceCountObj) {
  let finalResourceCounts = {}

  if (resourceCountObj && typeof resourceCountObj === 'object') {
    Object.keys(resourceCountObj).forEach(resourceType => {
      let resourceData = resourceCountObj[resourceType]

      // Extract clean resource name from type (remove API version prefixes)
      let resourceName = resourceType.split('.').pop() || resourceType

      let resourceInfo = {}

      // Get total count from summary
      if (resourceData.summary && resourceData.summary.count !== undefined) {
        resourceInfo.totalCount = resourceData.summary.count
      }

      // Get namespaced counts if they exist
      if (resourceData.namespaces && typeof resourceData.namespaces === 'object') {
        resourceInfo.namespaces = {}
        Object.keys(resourceData.namespaces).forEach(namespace => {
          let namespaceData = resourceData.namespaces[namespace]
          if (namespaceData.count !== undefined) {
            resourceInfo.namespaces[namespace] = namespaceData.count
          }
        })
      }

      // Only add to final object if we have some data
      if (resourceInfo.totalCount !== undefined || resourceInfo.namespaces) {
        finalResourceCounts[resourceName] = resourceInfo
      }
    })
  }

  return finalResourceCounts
}

export function getLocalClusterAccessibleDiagnosticsSchemas(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/schemas`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
  })

  return JSON.parse(response.body).data.map(s => s.id)
}

export function getLocalClusterSystemImages(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/management.cattle.io.rkek8ssystemimage`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/management.cattle.io.rkek8ssystemimage can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterEvents(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/event`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/event can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterK8sEvents(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/events.k8s.io.event`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/events.k8s.io.event can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterSettings(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/management.cattle.io.setting`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/management.cattle.io.setting can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterClusterRoles(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/rbac.authorization.k8s.io.clusterrole`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/rbac.authorization.k8s.io.clusterrole can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterCRDs(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/apiextensions.k8s.io.customresourcedefinition`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/apiextensions.k8s.io.customresourcedefinition can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterRoles(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/rbac.authorization.k8s.io.role`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/rbac.authorization.k8s.io.roles can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterRoleBindings(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/rbac.authorization.k8s.io.rolebinding`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/rbac.authorization.k8s.io.rolebinding can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterClusterRoleBindings(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/rbac.authorization.k8s.io.clusterrolebinding`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/rbac.authorization.k8s.io.clusterrolebinding can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterGlobalRoleBindings(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/management.cattle.io.globalrolebinding`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/management.cattle.io.globalrolebinding can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterRKEAddons(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/management.cattle.io.rkeaddon`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/management.cattle.io.rkeaddon can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterConfigMaps(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/configmap`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/configmap can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterServiceAccounts(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/serviceaccount`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/serviceaccount can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterSecrets(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/secret`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/secret can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterPods(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/pod`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/pod can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterRKEK8sServiceOptions(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/management.cattle.io.rkek8sserviceoption`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/management.cattle.io.rkek8sserviceoption can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterAPIServices(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/apiregistration.k8s.io.apiservice`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/apiregistration.k8s.io.apiservice can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterRoleTemplates(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/management.cattle.io.roletemplate`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/management.cattle.io.roletemplate can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterProjects(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/management.cattle.io.projects`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/management.cattle.io.projects can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterNamespaces(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/namespaces`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/namespaces can be queried': (r) => r.status === 200,
  })
  return response
}

export function processResourceTimings(baseUrl, cookies) {
  let timingsArr = {}

  let accessibleSchemas = getLocalClusterAccessibleDiagnosticsSchemas(baseUrl, cookies)

  // Get timing data for each resource type
  if (accessibleSchemas.includes("management.cattle.io.rkek8ssystemimage")) {
    let systemImageRes = getLocalClusterSystemImages(baseUrl, cookies)
    if (systemImageRes.status == 200 && systemImageRes.timings.duration) {
      timingsArr["management.cattle.io.rkek8ssystemimage"] = systemImageRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("event")) {
    let eventRes = getLocalClusterEvents(baseUrl, cookies)
    if (eventRes.status == 200 && eventRes.timings.duration) {
      timingsArr["event"] = eventRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("events.k8s.io.event")) {
    let k8sEventRes = getLocalClusterK8sEvents(baseUrl, cookies)
    if (k8sEventRes.status == 200 && k8sEventRes.timings.duration) {
      timingsArr["events.k8s.io.event"] = k8sEventRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("management.cattle.io.setting")) {
    let settingsRes = getLocalClusterSettings(baseUrl, cookies)
    if (settingsRes.status == 200 && settingsRes.timings.duration) {
      timingsArr["management.cattle.io.setting"] = settingsRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("rbac.authorization.k8s.io.clusterrole")) {
    let clusterRoleRes = getLocalClusterClusterRoles(baseUrl, cookies)
    if (clusterRoleRes.status == 200 && clusterRoleRes.timings.duration) {
      timingsArr["rbac.authorization.k8s.io.clusterrole"] = clusterRoleRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("apiextensions.k8s.io.customresourcedefinition")) {
    let crdRes = getLocalClusterCRDs(baseUrl, cookies)
    if (crdRes.status == 200 && crdRes.timings.duration) {
      timingsArr["apiextensions.k8s.io.customresourcedefinition"] = crdRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("rbac.authorization.k8s.io.role")) {
    let roleRes = getLocalClusterRoles(baseUrl, cookies)
    if (roleRes.status == 200 && roleRes.timings.duration) {
      timingsArr["rbac.authorization.k8s.io.role"] = roleRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("rbac.authorization.k8s.io.rolebinding")) {
    let roleBindingRes = getLocalClusterRoleBindings(baseUrl, cookies)
    if (roleBindingRes.status == 200 && roleBindingRes.timings.duration) {
      timingsArr["rbac.authorization.k8s.io.rolebinding"] = roleBindingRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("rbac.authorization.k8s.io.clusterrolebinding")) {
    let clusterRoleBindingRes = getLocalClusterClusterRoleBindings(baseUrl, cookies)
    if (clusterRoleBindingRes.status == 200 && clusterRoleBindingRes.timings.duration) {
      timingsArr["rbac.authorization.k8s.io.clusterrolebinding"] = clusterRoleBindingRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("management.cattle.io.globalrolebinding")) {
    let globalRoleBindingRes = getLocalClusterGlobalRoleBindings(baseUrl, cookies)
    if (globalRoleBindingRes.status == 200 && globalRoleBindingRes.timings.duration) {
      timingsArr["management.cattle.io.globalrolebinding"] = globalRoleBindingRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("management.cattle.io.rkeaddon")) {
    let rkeAddonRes = getLocalClusterRKEAddons(baseUrl, cookies)
    if (rkeAddonRes.status == 200 && rkeAddonRes.timings.duration) {
      timingsArr["management.cattle.io.rkeaddon"] = rkeAddonRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("configmap")) {
    let configMapRes = getLocalClusterConfigMaps(baseUrl, cookies)
    if (configMapRes.status == 200 && configMapRes.timings.duration) {
      timingsArr["configmap"] = configMapRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("serviceaccount")) {
    let serviceAccountRes = getLocalClusterServiceAccounts(baseUrl, cookies)
    if (serviceAccountRes.status == 200 && serviceAccountRes.timings.duration) {
      timingsArr["serviceaccount"] = serviceAccountRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("secret")) {
    let secretRes = getLocalClusterSecrets(baseUrl, cookies)
    if (secretRes.status == 200 && secretRes.timings.duration) {
      timingsArr["secret"] = secretRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("pod")) {
    let podRes = getLocalClusterPods(baseUrl, cookies)
    if (podRes.status == 200 && podRes.timings.duration) {
      timingsArr["pod"] = podRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("management.cattle.io.rkek8sserviceoption")) {
    let rkeK8sServiceOptionRes = getLocalClusterRKEK8sServiceOptions(baseUrl, cookies)
    if (rkeK8sServiceOptionRes.status == 200 && rkeK8sServiceOptionRes.timings.duration) {
      timingsArr["management.cattle.io.rkek8sserviceoption"] = rkeK8sServiceOptionRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("apiregistration.k8s.io.apiservice")) {
    let apiServiceRes = getLocalClusterAPIServices(baseUrl, cookies)
    if (apiServiceRes.status == 200 && apiServiceRes.timings.duration) {
      timingsArr["apiregistration.k8s.io.apiservice"] = apiServiceRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("management.cattle.io.roletemplate")) {
    let roleTemplateRes = getLocalClusterRoleTemplates(baseUrl, cookies)
    if (roleTemplateRes.status == 200 && roleTemplateRes.timings.duration) {
      timingsArr["management.cattle.io.roletemplate"] = roleTemplateRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("management.cattle.io.project")) {
    let projectsRes = getLocalClusterProjects(baseUrl, cookies)
    if (projectsRes.status == 200 && projectsRes.timings.duration) {
      timingsArr["management.cattle.io.project"] = projectsRes.timings.duration
    }
  }

  if (accessibleSchemas.includes("namespace")) {
    let namespacesRes = getLocalClusterNamespaces(baseUrl, cookies)
    if (namespacesRes.status == 200 && namespacesRes.timings.duration) {
      timingsArr["namespace"] = namespacesRes.timings.duration
    }
  }

  return timingsArr
}
