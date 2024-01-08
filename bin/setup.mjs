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
    getAppAddressesFor,
    importImage,
} from "./lib/common.mjs"
import {k6_run} from "./lib/k6.mjs";
import {install_rancher_monitoring} from "./lib/rancher_monitoring.mjs";

// Parameters
const RANCHER_VERSION = "2.7.9"
const RANCHER_CHART = `https://releases.rancher.com/server-charts/latest/rancher-${RANCHER_VERSION}.tgz`
const RANCHER_IMAGE_TAG = `v${RANCHER_VERSION}`
const CERT_MANAGER_CHART = "https://charts.jetstack.io/charts/cert-manager-v1.8.0.tgz"
const GRAFANA_CHART = "https://github.com/grafana/helm-charts/releases/download/grafana-6.56.5/grafana-6.56.5.tgz"

// Step 1: Terraform
run(`terraform -chdir=${q(terraformDir())} init -upgrade`)
run(`terraform -chdir=${q(terraformDir())} apply -auto-approve ${q(terraformVar())}`)
const clusters = runCollectingJSONOutput(`terraform -chdir=${q(terraformDir())} output -json`)["clusters"]["value"]

// Step 3: Helm charts
// tester cluster
const tester = clusters["tester"]
helm_install("mimir", dir("charts/mimir"), tester, "tester", {ingressClassName: tester["ingress_class_name"]})
helm_install("k6-files", dir("charts/k6-files"), tester, "tester", {})
helm_install("grafana-dashboards", dir("charts/grafana-dashboards"), tester, "tester", {})

const testerAddresses = getAppAddressesFor(tester)
const grafanaName = testerAddresses.localNetwork.name
const grafanaURL = testerAddresses.localNetwork.httpURL
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
        hosts: [grafanaName],
        ingressClassName: tester["ingress_class_name"],
    },
    env: {
        "GF_SERVER_ROOT_URL": `${grafanaURL}/grafana`,
        "GF_SERVER_SERVE_FROM_SUB_PATH": "true"
    },
    adminPassword: ADMIN_PASSWORD,
})

// upstream cluster
const upstream = clusters["upstream"]
importImage(`rancher/rancher:${RANCHER_IMAGE_TAG}`, upstream)
helm_install("cert-manager", CERT_MANAGER_CHART, upstream, "cert-manager", {installCRDs: true})

const BOOTSTRAP_PASSWORD = "admin"
const upstreamAddresses = getAppAddressesFor(upstream)
const rancherClusterNetworkName = upstreamAddresses.clusterNetwork.name
const rancherClusterNetworkURL = upstreamAddresses.clusterNetwork.httpsURL
helm_install("rancher", RANCHER_CHART, upstream, "cattle-system", {
    bootstrapPassword: BOOTSTRAP_PASSWORD,
    hostname: rancherClusterNetworkName,
    replicas: isK3d() ? 1 : 3,
    rancherImageTag: RANCHER_IMAGE_TAG,
    extraEnv: [{
        name: "CATTLE_SERVER_URL",
        value: rancherClusterNetworkURL
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
    },
})

const rancherName = upstreamAddresses.localNetwork.name
const rancherURL = upstreamAddresses.localNetwork.httpsURL
helm_install("rancher-ingress", dir("charts/rancher-ingress"), upstream, "default", {
    sans: [rancherName, rancherClusterNetworkName].filter(x => x),
    ingressClassName: upstream["ingress_class_name"],
})

const monitoringRestrictions = {
    nodeSelector: {monitoring: "true"},
    tolerations: [{key: "monitoring", operator: "Exists", effect: "NoSchedule"}],
}

const mimirClusterNetworkURL = testerAddresses.clusterNetwork.httpURL
install_rancher_monitoring(upstream, isK3d() ? {} : monitoringRestrictions, `${mimirClusterNetworkURL}/mimir/api/v1/push`)

helm_install("cgroups-exporter", dir("charts/cgroups-exporter"), upstream, "cattle-monitoring-system", {})

const kuf = `--kubeconfig=${upstream["kubeconfig"]}`
const cuf = `--context=${upstream["context"]}`
run(`kubectl wait deployment/rancher --namespace cattle-system --for condition=Available=true --timeout=1h ${q(kuf)} ${q(cuf)}`)


// Step 4: Import downstream clusters
const importedClusters = Object.entries(clusters).filter(([k,v]) => k.startsWith("downstream"))
const importedClusterNames = importedClusters.map(([name, cluster]) => name).join(",")

importImage(`rancher/rancher-agent:${RANCHER_IMAGE_TAG}`, ...importedClusters.map(([name, cluster]) => cluster))

k6_run(tester, { BASE_URL: rancherClusterNetworkURL, BOOTSTRAP_PASSWORD: BOOTSTRAP_PASSWORD, PASSWORD: ADMIN_PASSWORD, IMPORTED_CLUSTER_NAMES: importedClusterNames}, {}, "k6/rancher_setup.js")

for (const [name, cluster] of importedClusters) {
    const clusterId = await retryOnError(() =>
        runCollectingJSONOutput(`kubectl get -n fleet-default clusters.provisioning.cattle.io ${q(name)} -o json ${q(kuf)} ${q(cuf)}`)["status"]["clusterName"]
    )
    const token = runCollectingJSONOutput(`kubectl get -n ${q(clusterId)} clusterregistrationtoken.management.cattle.io default-token -o json ${q(kuf)} ${q(cuf)}`)["status"]["token"]

    const url = `${rancherURL}/v3/import/${token}_${clusterId}.yaml`
    const yaml = runCollectingOutput(`curl --insecure -fL ${q(url)}`)
    run(`kubectl apply -f - --kubeconfig=${q(cluster["kubeconfig"])} --context=${q(cluster["context"])}`, {input: yaml})
}

run(`kubectl wait clusters.management.cattle.io --all --for condition=ready=true --timeout=15m ${q(kuf)} ${q(cuf)}`)

if (importedClusters.length > 0) {
    run(`kubectl wait cluster.fleet.cattle.io --all --namespace fleet-default --for condition=ready=true --timeout=15m ${q(kuf)} ${q(cuf)}`)
}

for (const [_, cluster] of importedClusters) {
    install_rancher_monitoring(cluster, {})
}
