#!/usr/bin/env node
import {ADMIN_PASSWORD, dir, helm_install, run, runCollectingJSONOutput, runCollectingOutput} from "./lib/common.mjs"

// Parameters
const CERT_MANAGER_CHART = "https://charts.jetstack.io/charts/cert-manager-v1.8.0.tgz"
const OLD_RANCHER_CHART = "https://releases.rancher.com/server-charts/latest/rancher-2.7.1.tgz"
const NEW_RANCHER_CHART = "https://releases.rancher.com/server-charts/latest/rancher-2.7.3.tgz"
const RANCHER_MONITORING_CRD_CHART = "https://github.com/rancher/charts/raw/release-v2.7/assets/rancher-monitoring-crd/rancher-monitoring-crd-102.0.0%2Bup40.1.2.tgz"
const RANCHER_MONITORING_CHART = "https://github.com/rancher/charts/raw/release-v2.7/assets/rancher-monitoring/rancher-monitoring-102.0.0%2Bup40.1.2.tgz"

// Step 1: Terraform
run(`terraform -chdir=${dir("terraform")} init`)
run(`terraform -chdir=${dir("terraform")} apply -auto-approve`)
const tfOutput = runCollectingJSONOutput(`terraform -chdir=${dir("terraform")} output -json`)



// Step 2: Helm charts
// old cluster
const oldUpstreamCluster = tfOutput["old_upstream_cluster"]["value"]
helm_install("cert-manager", CERT_MANAGER_CHART, oldUpstreamCluster, "cert-manager", {installCRDs: true})

const BOOTSTRAP_PASSWORD = "admin"
const oldHostname = tfOutput["old_upstream_cluster_private_name"]["value"]
helm_install("rancher", OLD_RANCHER_CHART, oldUpstreamCluster, "cattle-system", {
    bootstrapPassword: BOOTSTRAP_PASSWORD,
    hostname: oldHostname,
    replicas: 1,
    extraEnv: [{
        name: "CATTLE_SERVER_URL",
        value: `https://${oldHostname}:443`
    }],
})

const oldUpstreamSAN = tfOutput["old_upstream_san"]["value"]
helm_install("rancher-ingress", dir("charts/rancher-ingress"), oldUpstreamCluster, "default", {
    san: oldUpstreamSAN,
})

helm_install("rancher-monitoring-crd", RANCHER_MONITORING_CRD_CHART, oldUpstreamCluster, "cattle-monitoring-system", {
    global: {
        cattle: {
            clusterId: "local",
            clusterName: "local",
            systemDefaultRegistry: "",
        }
    },
    systemDefaultRegistry: "",
})

helm_install("rancher-monitoring", RANCHER_MONITORING_CHART, oldUpstreamCluster, "cattle-monitoring-system", {
    alertmanager: { enabled:"false" },
    prometheus: {
        prometheusSpec: {
            evaluationInterval: "1m",
            retentionSize: "50GiB",
            scrapeInterval: "1m",
        }
    },
    global: {
        cattle: {
            clusterId: "local",
            clusterName: "local",
            systemDefaultRegistry: "",
        }
    },
    systemDefaultRegistry: "",
})

// new cluster
const newUpstreamCluster = tfOutput["new_upstream_cluster"]["value"]
helm_install("cert-manager", CERT_MANAGER_CHART, newUpstreamCluster, "cert-manager", {installCRDs: true})

const newHostname = tfOutput["new_upstream_cluster_private_name"]["value"]
helm_install("rancher", NEW_RANCHER_CHART, newUpstreamCluster, "cattle-system", {
    bootstrapPassword: BOOTSTRAP_PASSWORD,
    hostname: newHostname,
    replicas: 1,
    extraEnv: [{
        name: "CATTLE_SERVER_URL",
        value: `https://${newHostname}:443`
    }],
})

const newUpstreamSAN = tfOutput["new_upstream_san"]["value"]
helm_install("rancher-ingress", dir("charts/rancher-ingress"), newUpstreamCluster, "default", {
    san: newUpstreamSAN,
})

helm_install("rancher-monitoring-crd", RANCHER_MONITORING_CRD_CHART, newUpstreamCluster, "cattle-monitoring-system", {
    global: {
        cattle: {
            clusterId: "local",
            clusterName: "local",
            systemDefaultRegistry: "",
        }
    },
    systemDefaultRegistry: "",
})

helm_install("rancher-monitoring", RANCHER_MONITORING_CHART, newUpstreamCluster, "cattle-monitoring-system", {
    alertmanager: { enabled:"false" },
    prometheus: {
        prometheusSpec: {
            evaluationInterval: "1m",
            retentionSize: "50GiB",
            scrapeInterval: "1m",
        }
    },
    global: {
        cattle: {
            clusterId: "local",
            clusterName: "local",
            systemDefaultRegistry: "",
        }
    },
    systemDefaultRegistry: "",
})

const ouf = `--kubeconfig=${oldUpstreamCluster["kubeconfig"]} --context=${oldUpstreamCluster["context"]}`
run(`kubectl wait deployment/rancher --namespace cattle-system --for condition=Available=true --timeout=1h ${ouf}`)

const nuf = `--kubeconfig=${newUpstreamCluster["kubeconfig"]} --context=${newUpstreamCluster["context"]}`
run(`kubectl wait deployment/rancher --namespace cattle-system --for condition=Available=true --timeout=1h ${nuf}`)


// Step 3: Import downstream clusters
// old
const oldUpstreamPublicPort = tfOutput["old_upstream_public_port"]["value"]
const oldBaseUrl = `https://${oldUpstreamSAN}:${oldUpstreamPublicPort}`
const oldImportedClusters = tfOutput["old_downstream_clusters"]["value"]
const oldImportedClusterNames = oldImportedClusters.map(c => c["name"])
run(`k6 run -e BASE_URL=${oldBaseUrl} -e BOOTSTRAP_PASSWORD=${BOOTSTRAP_PASSWORD} -e PASSWORD=${ADMIN_PASSWORD} -e IMPORTED_CLUSTER_NAMES=${oldImportedClusterNames} ${dir("k6")}/rancher_setup.js`)

for (const i in oldImportedClusters) {
    const name = oldImportedClusters[i]["name"]
    const df = `--kubeconfig=${oldImportedClusters[i]["kubeconfig"]} --context=${oldImportedClusters[i]["context"]}`

    const clusterId = runCollectingJSONOutput(`kubectl get -n fleet-default cluster ${name} -o json ${ouf}`)["status"]["clusterName"]
    const token = runCollectingJSONOutput(`kubectl get -n ${clusterId} clusterregistrationtoken.management.cattle.io default-token -o json ${ouf}`)["status"]["token"]

    const url = `${oldBaseUrl}/v3/import/${token}_${clusterId}.yaml`
    const yaml = runCollectingOutput(`curl --insecure -fL ${url}`)
    run(`kubectl apply -f - ${df}`, {input: yaml})
}

// new
const newUpstreamPublicPort = tfOutput["new_upstream_public_port"]["value"]
const newBaseUrl = `https://${newUpstreamSAN}:${newUpstreamPublicPort}`
const newImportedClusters = tfOutput["new_downstream_clusters"]["value"]
const newImportedClusterNames = newImportedClusters.map(c => c["name"])
run(`k6 run -e BASE_URL=${newBaseUrl} -e BOOTSTRAP_PASSWORD=${BOOTSTRAP_PASSWORD} -e PASSWORD=${ADMIN_PASSWORD} -e IMPORTED_CLUSTER_NAMES=${newImportedClusterNames} ${dir("k6")}/rancher_setup.js`)

for (const i in newImportedClusters) {
    const name = newImportedClusters[i]["name"]
    const df = `--kubeconfig=${newImportedClusters[i]["kubeconfig"]} --context=${newImportedClusters[i]["context"]}`

    const clusterId = runCollectingJSONOutput(`kubectl get -n fleet-default cluster ${name} -o json ${nuf}`)["status"]["clusterName"]
    const token = runCollectingJSONOutput(`kubectl get -n ${clusterId} clusterregistrationtoken.management.cattle.io default-token -o json ${nuf}`)["status"]["token"]

    const url = `${newBaseUrl}/v3/import/${token}_${clusterId}.yaml`
    const yaml = runCollectingOutput(`curl --insecure -fL ${url}`)
    run(`kubectl apply -f - ${df}`, {input: yaml})
}

run(`kubectl wait clusters.management.cattle.io --all --for condition=ready=true --timeout=1h ${ouf}`)
run(`kubectl wait cluster.fleet.cattle.io --all --namespace fleet-default --for condition=ready=true --timeout=1h ${ouf}`)

run(`kubectl wait clusters.management.cattle.io --all --for condition=ready=true --timeout=1h ${nuf}`)
run(`kubectl wait cluster.fleet.cattle.io --all --namespace fleet-default --for condition=ready=true --timeout=1h ${nuf}`)

console.log(`*** All direct cluster access is already in your ~/.kube/config`)
console.log(`*** Access OLD Rancher: ${oldBaseUrl} (admin/adminadminadmin)`)
console.log(`*** Access NEW Rancher: ${newBaseUrl} (admin/adminadminadmin)`)
