#!/usr/bin/env node
import {
    ADMIN_PASSWORD,
    dir,
    terraformDir,
    terraformVar,
    helm_install,
    q,
    run,
    runCollectingJSONOutput,
    runCollectingOutput,
    isK3d,
    retryOnError,
} from "./lib/common.mjs"
import {k6_run} from "./lib/k6.mjs";
import {install_rancher_monitoring} from "./lib/rancher_monitoring.mjs";

// Parameters
const RANCHER_VERSION = "2.7.6"
const RANCHER_CHART = `https://releases.rancher.com/server-charts/latest/rancher-${RANCHER_VERSION}.tgz`
const RANCHER_IMAGE_TAG = `v${RANCHER_VERSION}`
const CERT_MANAGER_CHART = "https://charts.jetstack.io/charts/cert-manager-v1.8.0.tgz"
const GRAFANA_CHART = "https://github.com/grafana/helm-charts/releases/download/grafana-6.56.5/grafana-6.56.5.tgz"

// Step 1: Terraform
run(`terraform -chdir=${q(terraformDir())} init -upgrade`)
run(`terraform -chdir=${q(terraformDir())} apply -auto-approve ${q(terraformVar())}`)
const clusters = runCollectingJSONOutput(`terraform -chdir=${q(terraformDir())} output -json`)["clusters"]["value"]



// Step 2: Helm charts
// tester cluster
const tester = clusters["tester"]
helm_install("mimir", dir("charts/mimir"), tester, "tester", {})
helm_install("k6-files", dir("charts/k6-files"), tester, "tester", {})
helm_install("grafana-dashboards", dir("charts/grafana-dashboards"), tester, "tester", {})

const localTesterName = tester["local_name"]
helm_install("grafana", GRAFANA_CHART, tester, "tester", {
    datasources: {
        "datasources.yaml": {
            apiVersion: 1,
            datasources: [{
                name: "mimir",
                type: "prometheus",
                url: "http://mimir.tester:9009/mimir/prometheus",
                access: "proxy",
                isDefault: true
            }]
        }
    },
    dashboardProviders: {
        "dashboardproviders.yaml": {
            apiVersion: 1,
            providers: [{
                name: "default",
                folder: "",
                type: "file",
                disableDeletion: false,
                editable: true,
                options: {
                    path: "/var/lib/grafana/dashboards/default"
                }
            }]
        }
    },
    dashboardsConfigMaps: { "default": "grafana-dashboards" },
    ingress: {
        enabled: true,
        path: "/grafana",
        hosts: [localTesterName]
    },
    env: {
        "GF_SERVER_ROOT_URL": `http://${localTesterName}/grafana`,
        "GF_SERVER_SERVE_FROM_SUB_PATH": "true"
    },
    adminPassword: ADMIN_PASSWORD,
})

// upstream cluster
const upstream = clusters["upstream"]
helm_install("cert-manager", CERT_MANAGER_CHART, upstream, "cert-manager", {installCRDs: true})

const BOOTSTRAP_PASSWORD = "admin"
const privateUpstreamName = upstream["private_name"]
const privateRancherUrl = `https://${privateUpstreamName}`
helm_install("rancher", RANCHER_CHART, upstream, "cattle-system", {
    bootstrapPassword: BOOTSTRAP_PASSWORD,
    hostname: privateUpstreamName,
    replicas: isK3d() ? 1 : 3,
    rancherImageTag: RANCHER_IMAGE_TAG,
    extraEnv: [{
        name: "CATTLE_SERVER_URL",
        value: privateRancherUrl
    },
    {
        name: "CATTLE_PROMETHEUS_METRICS",
        value: "true"
    },
    {
        name: "CATTLE_DEV_MODE",
        value: "true"
    }],
    livenessProbe: {
        initialDelaySeconds: 30,
        periodSeconds: 3600
    }
})

const localUpstreamName = upstream["local_name"]
helm_install("rancher-ingress", dir("charts/rancher-ingress"), upstream, "default", {
    san: localUpstreamName,
})

const monitoringRestrictions = {
    nodeSelector: {monitoring: "true"},
    tolerations: [{key: "monitoring", operator: "Exists", effect: "NoSchedule"}],
}

install_rancher_monitoring(upstream, isK3d() ? {} : monitoringRestrictions, `http://${tester["private_name"]}/mimir/api/v1/push`)

helm_install("cgroups-exporter", dir("charts/cgroups-exporter"), upstream, "cattle-monitoring-system", {})

const kuf = `--kubeconfig=${upstream["kubeconfig"]}`
const cuf = `--context=${upstream["context"]}`
run(`kubectl wait deployment/rancher --namespace cattle-system --for condition=Available=true --timeout=1h ${q(kuf)} ${q(cuf)}`)


// Step 3: Import downstream clusters
const localRancherUrl = `https://${localUpstreamName}:${upstream["local_https_port"]}`
const importedClusters = Object.entries(clusters).filter(([k,v]) => k.startsWith("downstream"))
const importedClusterNames = importedClusters.map(([name, cluster]) => name).join(",")
k6_run(tester, { BASE_URL: privateRancherUrl, BOOTSTRAP_PASSWORD: BOOTSTRAP_PASSWORD, PASSWORD: ADMIN_PASSWORD, IMPORTED_CLUSTER_NAMES: importedClusterNames}, {}, "k6/rancher_setup.js")

for (const [name, cluster] of importedClusters) {
    const clusterId = await retryOnError(() =>
        runCollectingJSONOutput(`kubectl get -n fleet-default cluster ${q(name)} -o json ${q(kuf)} ${q(cuf)}`)["status"]["clusterName"]
    )
    const token = runCollectingJSONOutput(`kubectl get -n ${q(clusterId)} clusterregistrationtoken.management.cattle.io default-token -o json ${q(kuf)} ${q(cuf)}`)["status"]["token"]

    const url = `${localRancherUrl}/v3/import/${token}_${clusterId}.yaml`
    const yaml = runCollectingOutput(`curl --insecure -fL ${q(url)}`)
    run(`kubectl apply -f - --kubeconfig=${q(cluster["kubeconfig"])} --context=${q(cluster["context"])}`, {input: yaml})
}

run(`kubectl wait clusters.management.cattle.io --all --for condition=ready=true --timeout=1h ${q(kuf)} ${q(cuf)}`)

if (importedClusters.length > 0) {
    run(`kubectl wait cluster.fleet.cattle.io --all --namespace fleet-default --for condition=ready=true --timeout=1h ${q(kuf)} ${q(cuf)}`)
}


// Output access details
console.log("*** ACCESS DETAILS")
console.log()

console.log(`*** UPSTREAM CLUSTER`)
console.log(`    Rancher UI: https://${upstream["local_name"]}:${upstream["local_https_port"]} (admin/${ADMIN_PASSWORD})`)
console.log(`    Kubernetes API:`)
console.log(`export KUBECONFIG=${q(upstream["kubeconfig"])}`)
console.log(`kubectl config use-context ${q(upstream["context"])}`)
for (const [node, command] of Object.entries(upstream["node_access_commands"])) {
    console.log(`    Node ${node}: ${q(command)}`)
}
console.log()

for (const [name, downstream] of importedClusters) {
    console.log(`*** ${name.toUpperCase()} CLUSTER`)
    console.log(`    Kubernetes API:`)
    console.log(`export KUBECONFIG=${q(downstream["kubeconfig"])}`)
    console.log(`kubectl config use-context ${q(downstream["context"])}`)
    for (const [node, command] of Object.entries(downstream["node_access_commands"])) {
        console.log(`    Node ${node}: ${q(command)}`)
    }
    console.log()
}

console.log(`*** TESTER CLUSTER`)
console.log(`    Grafana UI: http://${tester["local_name"]}:${tester["local_http_port"]}/grafana/d/a1508c35-b2e6-47f4-94ab-fec400d1c243/test-results?orgId=1&refresh=5s&from=now-30m&to=now (admin/${ADMIN_PASSWORD})`)
for (const [node, command] of Object.entries(tester["node_access_commands"])) {
    console.log(`    Node ${node}: ${q(command)}`)
}
console.log()
