import { check, fail, sleep } from 'k6'
import { Trend, Gauge } from 'k6/metrics';
import http from 'k6/http'


// Resource count tracking metrics (using Gauges since they represent current state)
export const totalEventsGauge = new Gauge('cluster_events_total')
export const totalK8sEventsGauge = new Gauge('cluster_k8sevents_total')
export const totalSettingsGauge = new Gauge('cluster_settings_total')
export const totalClusterRolesGauge = new Gauge('cluster_clusterroles_total')
export const totalCRDsGauge = new Gauge('cluster_crds_total')
export const totalRolesGauge = new Gauge('roles_total')
export const totalRoleBindingsGauge = new Gauge('cluster_rolebindings_total')
export const totalClusterRoleBindingsGauge = new Gauge('cluster_clusterrolebindings_total')
export const totalGlobalRoleBindingsGauge = new Gauge('cluster_globalrolebindings_total')
export const totalConfigMapsGauge = new Gauge('cluster_configmaps_total')
export const totalServiceAccountsGauge = new Gauge('cluster_serviceaccounts_total')
export const totalSecretsGauge = new Gauge('cluster_secrets_total')
export const totalPodsGauge = new Gauge('cluster_pods_total')
export const totalDeploymentsGauge = new Gauge('cluster_deployments_total')
export const totalServicesGauge = new Gauge('cluster_services_total')
export const totalAPIServicesGauge = new Gauge('cluster_apiservices_total')
export const totalRoleTemplatesGauge = new Gauge('cluster_roletemplates_total')
export const totalProjectsGauge = new Gauge('cluster_projects_total')
export const totalNamespacesGauge = new Gauge('cluster_namespaces_total')

// API response time tracking metrics
export const eventAPITime = new Trend('api_event_duration_ms')
export const k8sEventAPITime = new Trend('api_k8sevent_duration_ms')
export const settingsAPITime = new Trend('api_setting_duration_ms')
export const clusterRoleAPITime = new Trend('api_clusterrole_duration_ms')
export const crdAPITime = new Trend('api_crd_duration_ms')
export const roleAPITime = new Trend('api_role_duration_ms')
export const roleBindingAPITime = new Trend('api_rolebinding_duration_ms')
export const clusterRoleBindingAPITime = new Trend('api_clusterrolebinding_duration_ms')
export const globalRoleBindingAPITime = new Trend('api_globalrolebinding_duration_ms')
export const configMapAPITime = new Trend('api_configmap_duration_ms')
export const serviceAccountAPITime = new Trend('api_serviceaccount_duration_ms')
export const secretAPITime = new Trend('api_secret_duration_ms')
export const podAPITime = new Trend('api_pod_duration_ms')
export const deploymentAPITime = new Trend('api_deployment_duration_ms')
export const serviceAPITime = new Trend('api_service_duration_ms')
export const apiServiceAPITime = new Trend('api_apiservice_duration_ms')
export const roleTemplateAPITime = new Trend('api_roletemplate_duration_ms')
export const projectAPITime = new Trend('api_project_duration_ms')
export const namespaceAPITime = new Trend('api_namespace_duration_ms')

export const resourceCount = new Gauge('resource_count')
export const namespaceResourceDensity = new Trend('namespace_resource_density');
export const resourceDistribution = new Trend('resource_distribution')

export const timingTag = { timing: "yes" }
export const clusterScopeTag = { scope: "cluster" }
export const namespaceScopeTag = { scope: "namespace" }
export const localClusterTag = { cluster: "local" }

export const metrics = [
  {
    key: "event", label: "Events",
    gauge: totalEventsGauge, trend: eventAPITime,
    fetcher: getLocalClusterEvents
  },
  {
    key: "events.k8s.io.event", label: "K8s Events",
    gauge: totalK8sEventsGauge, trend: k8sEventAPITime,
    fetcher: getLocalClusterK8sEvents
  },
  {
    key: "management.cattle.io.setting", label: "Settings",
    gauge: totalSettingsGauge, trend: settingsAPITime,
    fetcher: getLocalClusterSettings
  },
  {
    key: "rbac.authorization.k8s.io.clusterrole", label: "Cluster Roles",
    gauge: totalClusterRolesGauge, trend: clusterRoleAPITime,
    fetcher: getLocalClusterRoles
  },
  {
    key: "apiextensions.k8s.io.customresourcedefinition", label: "CRDs",
    gauge: totalCRDsGauge, trend: crdAPITime,
    fetcher: getLocalClusterCRDs
  },
  {
    key: "rbac.authorization.k8s.io.role", label: "Roles",
    gauge: totalRolesGauge, trend: roleAPITime,
    fetcher: getLocalClusterRoles
  },
  {
    key: "rbac.authorization.k8s.io.role", label: "RoleBindings",
    gauge: totalRoleBindingsGauge, trend: roleBindingAPITime,
    fetcher: getLocalClusterRoleBindings
  },
  {
    key: "rbac.authorization.k8s.io.clusterrolebinding", label: "ClusterRoleBindings",
    gauge: totalClusterRoleBindingsGauge, trend: clusterRoleBindingAPITime,
    fetcher: getLocalClusterClusterRoleBindings
  },
  {
    key: "management.cattle.io.globalrolebinding", label: "GlobalRoleBindings",
    gauge: totalGlobalRoleBindingsGauge, trend: globalRoleBindingAPITime,
    fetcher: getLocalClusterGlobalRoleBindings
  },
  {
    key: "configmap", label: "ConfigMaps",
    gauge: totalConfigMapsGauge, trend: configMapAPITime,
    fetcher: getLocalClusterConfigMaps
  },
  {
    key: "serviceaccount", label: "ServiceAccounts",
    gauge: totalServiceAccountsGauge, trend: serviceAccountAPITime,
    fetcher: getLocalClusterServiceAccounts
  },
  {
    key: "secret", label: "Secrets",
    gauge: totalSecretsGauge, trend: secretAPITime,
    fetcher: getLocalClusterSecrets
  },
  {
    key: "pod", label: "Pods",
    gauge: totalPodsGauge, trend: podAPITime,
    fetcher: getLocalClusterPods
  },
  {
    key: "apps.deployment", label: "Deployments",
    gauge: totalDeploymentsGauge, trend: deploymentAPITime,
    fetcher: getLocalClusterDeployments
  },
  {
    key: "service", label: "Services",
    gauge: totalServicesGauge, trend: serviceAPITime,
    fetcher: getLocalClusterServices
  },
  {
    key: "apiregistration.k8s.io.apiservice", label: "APIServices",
    gauge: totalAPIServicesGauge, trend: apiServiceAPITime,
    fetcher: getLocalClusterAPIServices
  },
  {
    key: "management.cattle.io.roletemplate", label: "RoleTemplates",
    gauge: totalRoleTemplatesGauge, trend: roleTemplateAPITime,
    fetcher: getLocalClusterRoleTemplates
  },
  {
    key: "management.cattle.io.project", label: "Projects",
    gauge: totalProjectsGauge, trend: projectAPITime,
    fetcher: getLocalClusterProjects
  },
  {
    key: "namespace", label: "Namespaces",
    gauge: totalNamespacesGauge, trend: namespaceAPITime,
    fetcher: getLocalClusterNamespaces
  },
];

export function getLocalClusterAccessibleDiagnosticsSchemas(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/schemas`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
  })

  return JSON.parse(response.body).data.map(s => s.id)
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

export function getLocalClusterDeployments(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/apps.deployment`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/apps.deployment can be queried': (r) => r.status === 200,
  })
  return response
}

export function getLocalClusterServices(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/k8s/clusters/local/v1/service`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
    tags: timingTag
  })
  check(response, {
    'k8s/clusters/local/v1/service can be queried': (r) => r.status === 200,
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
        finalResourceCounts[resourceType] = resourceInfo
      }
    })
  }

  return finalResourceCounts
}

export function processResourceTimings(baseUrl, cookies) {
  console.log('Processing resource timings');
  const timingsArr = {};
  const schemas = getLocalClusterAccessibleDiagnosticsSchemas(baseUrl, cookies);

  metrics.forEach(({ key, fetcher }) => {
    if (schemas.includes(key)) {
      const res = fetcher(baseUrl, cookies);
      if (res.status === 200 && res.timings?.duration != null) {
        timingsArr[key] = res.timings.duration;
      }
    }
  });

  return timingsArr;
}
