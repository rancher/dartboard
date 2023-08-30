#!/usr/bin/env node
import {
    ADMIN_PASSWORD,
    dir,
    terraformDir,
    helm_install,
    q,
    runCollectingJSONOutput, runCollectingOutput, run,
} from "./lib/common.mjs"
import {k6_run} from "./lib/k6.mjs"

// Refresh k6 files on the tester cluster
const clusters = runCollectingJSONOutput(`terraform -chdir=${terraformDir()} output -json`)["clusters"]["value"]
const tester = clusters["tester"]
helm_install("k6-files", dir("charts/k6-files"), tester, "tester", {})

const commit = runCollectingOutput("git rev-parse --short HEAD").trim()

const upstream = clusters["upstream"]
const kuf = `--kubeconfig=${upstream["kubeconfig"]}`
const cuf = `--context=${upstream["context"]}`
const downstream = clusters["upstream"]
const kdf = `--kubeconfig=${downstream["kubeconfig"]}`
const cdf = `--context=${downstream["context"]}`

const downstreamClusterId = runCollectingJSONOutput(`kubectl get -o json ${q(kuf)} ${q(cuf)} -n fleet-default cluster ${downstream["name"]}`)["status"]["clusterName"]

for (const tag of ["v2.7.5", "improved"]) {
    run(`kubectl set image -n cattle-system deployment/rancher rancher=rancher/rancher:${tag} ${q(kuf)} ${q(cuf)}`)
    run(`kubectl rollout status --watch --timeout=3600s -n cattle-system deployment/rancher ${q(kuf)} ${q(cuf)}`)
    run(`kubectl set image -n cattle-system deployment/cattle-cluster-agent cluster-register=rancher/rancher-agent:${tag} ${q(kdf)} ${q(cdf)}`)
    run(`kubectl rollout status --watch --timeout=3600s -n cattle-system deployment/cattle-cluster-agent ${q(kdf)} ${q(cdf)}`)

    // HACK: allow 5 more minutes for Steve to start up on the remote cluster
    // this can be removed with a good way to detect the "Steve auth startup complete" log message
    await sleep(5*60)

    for (const count of [100, 400, 1600, 6400]) {
        for (const cluster of [upstream, downstream]) {
            k6_run(tester,
                { BASE_URL: `https://${cluster["private_name"]}:6443`, KUBECONFIG: cluster["kubeconfig"], CONTEXT: cluster["context"], CONFIG_MAP_COUNT: count, SECRET_COUNT: 1},
                {commit: commit, cluster: cluster["context"], test: "create_load.mjs", ConfigMaps: count, Secrets: 1},
                "k6/create_k8s_resources.js", true
            )
        }

        for (const test of ["load_steve_k8s_pagination", "load_steve_new_pagination"]) {
            for (const cluster of [upstream, downstream]) {

                const clusterId = cluster == upstream ? "local" : downstreamClusterId
                // warmup
                k6_run(tester,
                    { BASE_URL: `https://${cluster["private_name"]}:443`, USERNAME: "admin", PASSWORD: ADMIN_PASSWORD, VUS: 1, PER_VU_ITERATIONS: 5, CLUSTER: clusterId },
                    {commit: commit, cluster: clusterId, test: "${test}.js", ConfigMaps: count},
                    "k6/${test}.js"
                )

                // test + record
                k6_run(tester,
                    { BASE_URL: `https://${cluster["private_name"]}:443`, USERNAME: "admin", PASSWORD: ADMIN_PASSWORD, VUS: 10, PER_VU_ITERATIONS: 30, CLUSTER: clusterId },
                    {commit: commit, cluster: clusterId, test: "${test}.js", ConfigMaps: count},
                    "k6/${test}.js", true
                )
            }
        }
    }
}
