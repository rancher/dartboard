#!/usr/bin/env node
import {ADMIN_PASSWORD, dir, helm_install, q, run, runCollectingJSONOutput, runCollectingOutput} from "./lib/common.mjs"
import {k6_run} from "./lib/k6.mjs";

// Parameters
const CERT_MANAGER_CHART = "https://charts.jetstack.io/charts/cert-manager-v1.8.0.tgz"
const RANCHER_CHART = "https://releases.rancher.com/server-charts/latest/rancher-2.7.2.tgz"
const RANCHER_MONITORING_CRD_CHART = "https://github.com/rancher/charts/raw/release-v2.7/assets/rancher-monitoring-crd/rancher-monitoring-crd-102.0.0%2Bup40.1.2.tgz"
const RANCHER_MONITORING_CHART = "https://github.com/rancher/charts/raw/release-v2.7/assets/rancher-monitoring/rancher-monitoring-102.0.0%2Bup40.1.2.tgz"

// Step 1: Terraform
run(`terraform -chdir=${q(dir("terraform"))} init`)
run(`terraform -chdir=${q(dir("terraform"))} apply -auto-approve`)
const tfOutput = runCollectingJSONOutput(`terraform -chdir=${q(dir("terraform"))} output -json`)



// Step 2: Helm charts
const upstreamCluster = tfOutput["upstream_cluster"]["value"]
helm_install("cert-manager", CERT_MANAGER_CHART, upstreamCluster, "cert-manager", {installCRDs: true})

const BOOTSTRAP_PASSWORD = "admin"
const hostname = tfOutput["upstream_cluster_private_name"]["value"]
helm_install("rancher", RANCHER_CHART, upstreamCluster, "cattle-system", {
    bootstrapPassword: BOOTSTRAP_PASSWORD,
    hostname: hostname,
    replicas: 1,
    extraEnv: [{
        name: "CATTLE_SERVER_URL",
        value: `https://${hostname}:443`
    }],
})

const upstreamSAN = tfOutput["upstream_san"]["value"]
helm_install("rancher-ingress", dir("charts/rancher-ingress"), upstreamCluster, "default", {
    san: upstreamSAN,
})

helm_install("rancher-monitoring-crd", RANCHER_MONITORING_CRD_CHART, upstreamCluster, "cattle-monitoring-system", {
    global: {
        cattle: {
            clusterId: "local",
            clusterName: "local",
            systemDefaultRegistry: "",
        }
    },
    systemDefaultRegistry: "",
})

const monitoringRestrictions = {
    nodeSelector: {monitoring: "true"},
    tolerations: [{key: "monitoring", operator: "Exists", effect: "NoSchedule"}],
}
helm_install("mimir", dir("charts/mimir"), upstreamCluster, "cattle-monitoring-system", monitoringRestrictions)

helm_install("rancher-monitoring", RANCHER_MONITORING_CHART, upstreamCluster, "cattle-monitoring-system", {
    alertmanager: { enabled:"false" },
    grafana: {
        nodeSelector: {monitoring: "true"},
        tolerations: [{key: "monitoring", operator: "Exists", effect: "NoSchedule"}],
        additionalDataSources: [{
            name: 'mimir',
            type: 'prometheus',
            url: 'http://mimir:9009/prometheus',
        }]
    },
    prometheus: {
        prometheusSpec: {
            evaluationInterval: "1m",
            nodeSelector: {monitoring: "true"},
            tolerations: [{key: "monitoring", operator: "Exists", effect: "NoSchedule"}],
            resources: {limits: {memory: "5000Mi"}},
            retentionSize: "50GiB",
            scrapeInterval: "1m",

            // configure writing metrics to mimir
            remoteWrite: [{
                url: "http://mimir:9009/api/v1/push",
                writeRelabelConfigs: [
                    // drop all metrics except for the ones matching regex
                    {
                        sourceLabels: ['__name__'],
                        regex: "node_namespace_pod_container.*",
                        action: "keep",
                    },
                    // add a testsuite-commit label to all metrics
                    {
                        targetLabel: 'testsuite_commit',
                        replacement: runCollectingOutput("git rev-parse --short HEAD"),
                        action: "replace",
                    }]
            }]
        }
    },
    "prometheus-adapter": monitoringRestrictions,
    "kube-state-metrics": monitoringRestrictions,
    prometheusOperator: monitoringRestrictions,
    global: {
        cattle: {
            clusterId: "local",
            clusterName: "local",
            systemDefaultRegistry: "",
        }
    },
    systemDefaultRegistry: "",
})

helm_install("k6-files", dir("charts/k6-files"), upstreamCluster, "cattle-monitoring-system", monitoringRestrictions)

const kuf = `--kubeconfig=${upstreamCluster["kubeconfig"]}`
const cuf = `--context=${q(upstreamCluster["context"])}`
run(`kubectl wait deployment/rancher --namespace cattle-system --for condition=Available=true --timeout=1h ${q(kuf)} ${q(cuf)}`)



// Step 3: Import downstream clusters
const upstreamPublicPort = tfOutput["upstream_public_port"]["value"]
const importedClusters = tfOutput["downstream_clusters"]["value"]
const importedClusterNames = importedClusters.map(c => c["name"]).join(",")
k6_run({VUS: 10, PER_VU_ITERATIONS: 30, BOOTSTRAP_PASSWORD: BOOTSTRAP_PASSWORD, PASSWORD: ADMIN_PASSWORD, IMPORTED_CLUSTER_NAMES: importedClusterNames}, "k6/rancher_setup.js")

const baseUrl = `https://${upstreamSAN}:${upstreamPublicPort}`
for (const i in importedClusters) {
    const name = importedClusters[i]["name"]
    const kdf = `--kubeconfig=${q(importedClusters[i]["kubeconfig"])}`
    const cdf = `--context=${q(importedClusters[i]["context"])}`

    const clusterId = runCollectingJSONOutput(`kubectl get -n fleet-default cluster ${q(name)} -o json ${q(kuf)} ${q(cuf)}`)["status"]["clusterName"]
    const token = runCollectingJSONOutput(`kubectl get -n ${q(clusterId)} clusterregistrationtoken.management.cattle.io default-token -o json ${q(kuf)} ${q(cuf)}`)["status"]["token"]

    const url = `${baseUrl}/v3/import/${token}_${clusterId}.yaml`
    const yaml = runCollectingOutput(`curl --insecure -fL ${q(url)}`)
    run(`kubectl apply -f - ${q(kdf)} ${q(cdf)}`, {input: yaml})
}

run(`kubectl wait clusters.management.cattle.io --all --for condition=ready=true --timeout=1h ${q(kuf)} ${q(cuf)}`)
run(`kubectl wait cluster.fleet.cattle.io --all --namespace fleet-default --for condition=ready=true --timeout=1h ${q(kuf)} ${q(cuf)}`)
