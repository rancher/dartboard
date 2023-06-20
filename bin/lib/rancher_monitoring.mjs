import {helm_install} from "./common.mjs";

const RANCHER_MONITORING_CHART = "https://github.com/rancher/charts/raw/release-v2.7/assets/rancher-monitoring/rancher-monitoring-102.0.0%2Bup40.1.2.tgz"
const RANCHER_MONITORING_CRD_CHART = "https://github.com/rancher/charts/raw/release-v2.7/assets/rancher-monitoring-crd/rancher-monitoring-crd-102.0.0%2Bup40.1.2.tgz"

export function install_rancher_monitoring(cluster, monitoringRestrictions, mimirUrl = null) {
    helm_install("rancher-monitoring-crd", RANCHER_MONITORING_CRD_CHART, cluster, "cattle-monitoring-system", {
        global: {
            cattle: {
                clusterId: "local",
                clusterName: "local",
                systemDefaultRegistry: "",
            }
        },
        systemDefaultRegistry: "",
    })

    helm_install("rancher-monitoring", RANCHER_MONITORING_CHART, cluster, "cattle-monitoring-system", {
        alertmanager: {enabled: "false"},
        grafana: monitoringRestrictions,
        prometheus: {
            prometheusSpec: {
                evaluationInterval: "1m",
                nodeSelector: monitoringRestrictions["nodeSelector"],
                tolerations: monitoringRestrictions["tolerations"],
                resources: {limits: {memory: "5000Mi"}},
                retentionSize: "50GiB",
                scrapeInterval: "1m",

                // configure scraping from cgroups-exporter
                additionalScrapeConfigs: [
                    {
                        job_name: "node-cgroups-exporter",
                        honor_labels: false,
                        kubernetes_sd_configs: [{
                                "role": "node"
                        }],
                        scheme: "http",
                        relabel_configs: [
                            {
                                action: "labelmap",
                                regex: "__meta_kubernetes_node_label_(.+)",
                            },
                            {
                                source_labels: ["__address__"],
                                action: "replace",
                                target_label: "__address__",
                                regex: "([^:;]+):(\\d+)",
                                replacement: "${1}:9753"
                            },
                            {
                                source_labels: ["__meta_kubernetes_node_name"],
                                action: "keep",
                                regex: ".*"
                            },
                            {
                                source_labels: ["__meta_kubernetes_node_name"],
                                action: "replace",
                                target_label: "node",
                                regex: "(.*)",
                                replacement: "${1}"
                            }
                        ]
                    }
                ],

                // configure writing metrics to mimir
                remoteWrite: mimirUrl != null ? [{
                    url: mimirUrl,
                    writeRelabelConfigs: [
                        // drop all metrics except for the ones matching regex
                        {
                            sourceLabels: ['__name__'],
                            regex: "(node_namespace_pod_container|node_load|node_memory|node_network_receive_bytes_total|container_network_receive_bytes_total|cgroups_).*",
                            action: "keep",
                        }
                    ]
                }] : [],
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
}
