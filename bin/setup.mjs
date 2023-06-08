#!/usr/bin/env node
import {ADMIN_PASSWORD, dir, helm_install, q, run, runCollectingJSONOutput, runCollectingOutput} from "./lib/common.mjs"
import {k6_run} from "./lib/k6.mjs";
import {install_rancher_monitoring} from "./lib/rancher_monitoring.mjs";

// Parameters
const CERT_MANAGER_CHART = "https://charts.jetstack.io/charts/cert-manager-v1.8.0.tgz"
const RANCHER_CHART = "https://releases.rancher.com/server-charts/latest/rancher-2.7.4.tgz"
const GRAFANA_CHART = "https://github.com/grafana/helm-charts/releases/download/grafana-6.56.5/grafana-6.56.5.tgz"

// Step 1: Terraform
run(`terraform -chdir=${q(dir("terraform"))} init -upgrade`)
run(`terraform -chdir=${q(dir("terraform"))} apply -auto-approve`)
const clusters = runCollectingJSONOutput(`terraform -chdir=${q(dir("terraform"))} output -json`)["clusters"]["value"]



// Step 2: Helm charts
// tester cluster
const tester = clusters["tester"]
const testerSAN = tester["san"]
helm_install("mimir", dir("charts/mimir"), tester, "tester", { san: testerSAN })
helm_install("k6-files", dir("charts/k6-files"), tester, "tester", {})
helm_install("grafana-dashboards", dir("charts/grafana-dashboards"), tester, "tester", {})

const MIMIR_URL = "http://mimir.tester:9009/mimir"
helm_install("grafana", GRAFANA_CHART, tester, "tester", {
    datasources: {
        "datasources.yaml": {
            apiVersion: 1,
            datasources: [{
                name: "mimir",
                type: "prometheus",
                url: MIMIR_URL + "/prometheus",
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
        hosts: [testerSAN]
    },
    env: {
        "GF_SERVER_ROOT_URL": `http://${testerSAN}/grafana`,
        "GF_SERVER_SERVE_FROM_SUB_PATH": "true"
    },
    adminPassword: ADMIN_PASSWORD,
})

// upstream cluster
const upstream = clusters["upstream"]
helm_install("cert-manager", CERT_MANAGER_CHART, upstream, "cert-manager", {installCRDs: true})

const BOOTSTRAP_PASSWORD = "admin"
const upstreamPrivateName = upstream["private_name"]
const privateRancherUrl = `https://${upstreamPrivateName}`
helm_install("rancher", RANCHER_CHART, upstream, "cattle-system", {
    bootstrapPassword: BOOTSTRAP_PASSWORD,
    hostname: upstreamPrivateName,
    replicas: 1,
    extraEnv: [{
        name: "CATTLE_SERVER_URL",
        value: privateRancherUrl
    }],
})

const upstreamSAN = upstream["san"]
helm_install("rancher-ingress", dir("charts/rancher-ingress"), upstream, "default", {
    san: upstreamSAN,
})

const monitoringRestrictions = {
    nodeSelector: {monitoring: "true"},
    tolerations: [{key: "monitoring", operator: "Exists", effect: "NoSchedule"}],
}
const testerPrivateName = tester["private_name"]

install_rancher_monitoring(upstream, monitoringRestrictions, `http://${testerPrivateName}/mimir/api/v1/push`)

const kuf = `--kubeconfig=${upstream["kubeconfig"]}`
const cuf = `--context=${upstream["context"]}`
run(`kubectl wait deployment/rancher --namespace cattle-system --for condition=Available=true --timeout=1h ${q(kuf)} ${q(cuf)}`)


// Step 3: Import downstream clusters
const upstreamPublicPort = upstream["public_https_port"]
const publicRancherUrl = `https://${upstreamSAN}:${upstreamPublicPort}`
const importedClusters = Object.entries(clusters).filter(([k,v]) => k.startsWith("downstream"))
const importedClusterNames = importedClusters.map(([name, cluster]) => name).join(",")
k6_run(tester, { BASE_URL: privateRancherUrl, BOOTSTRAP_PASSWORD: BOOTSTRAP_PASSWORD, PASSWORD: ADMIN_PASSWORD, IMPORTED_CLUSTER_NAMES: importedClusterNames}, {}, "k6/rancher_setup.js")

for (const [name, cluster] of importedClusters) {
    const clusterId = runCollectingJSONOutput(`kubectl get -n fleet-default cluster ${q(name)} -o json ${q(kuf)} ${q(cuf)}`)["status"]["clusterName"]
    const token = runCollectingJSONOutput(`kubectl get -n ${q(clusterId)} clusterregistrationtoken.management.cattle.io default-token -o json ${q(kuf)} ${q(cuf)}`)["status"]["token"]

    const url = `${publicRancherUrl}/v3/import/${token}_${clusterId}.yaml`
    const yaml = runCollectingOutput(`curl --insecure -fL ${q(url)}`)
    run(`kubectl apply -f - --kubeconfig=${q(cluster["kubeconfig"])} --context=${q(cluster["context"])}`, {input: yaml})
}

run(`kubectl wait clusters.management.cattle.io --all --for condition=ready=true --timeout=1h ${q(kuf)} ${q(cuf)}`)
run(`kubectl wait cluster.fleet.cattle.io --all --namespace fleet-default --for condition=ready=true --timeout=1h ${q(kuf)} ${q(cuf)}`)

for (const [_, cluster] of importedClusters) {
    install_rancher_monitoring(cluster, {})
}
