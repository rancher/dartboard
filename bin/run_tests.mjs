#!/usr/bin/env node
import {
    ADMIN_PASSWORD,
    dir,
    helm_install,
    q,
    run,
    runCollectingJSONOutput,
    runCollectingOutput,
    sleep
} from "./lib/common.mjs"
import {k6_run} from "./lib/k6.mjs";

const clusters = runCollectingJSONOutput(`terraform -chdir=${dir("terraform")} output -json`)["clusters"]["value"]

const upstream = clusters["upstream"]
const kuf = `--kubeconfig=${upstream["kubeconfig"]}`
const cuf = `--context=${upstream["context"]}`
const downstream = clusters["downstream"]
const kdf = `--kubeconfig=${downstream["kubeconfig"]}`
const cdf = `--context=${downstream["context"]}`
const downstreamId = runCollectingJSONOutput(`kubectl get -o json ${q(kuf)} ${q(cuf)} -n fleet-default cluster ${downstream["name"]}`)["status"]["clusterName"]

const tester = clusters["tester"]
const upstreamPrivateName = upstream["private_name"]
const privateRancherUrl = `https://${upstreamPrivateName}`

const commit = runCollectingOutput("git rev-parse --short HEAD").trim()

helm_install("k6-files", dir("charts/k6-files"), tester, "tester", {})

for (const tag of ["baseline", "vai"]) {
    run(`kubectl set image -n cattle-system deployment/rancher rancher=rancher/rancher:${q(tag)} ${q(kuf)} ${q(cuf)}`)
    run(`kubectl rollout status --watch --timeout=3600s -n cattle-system deployment/rancher ${q(kuf)} ${q(cuf)}`)
    run(`kubectl set image -n cattle-system deployment/cattle-cluster-agent cluster-register=rancher/rancher-agent:${q(tag)} ${q(kdf)} ${q(cdf)}`)
    run(`kubectl rollout status --watch --timeout=3600s -n cattle-system deployment/cattle-cluster-agent ${q(kdf)} ${q(cdf)}`)

    // HACK: allow 5 more minutes for Steve to start up on the remote cluster
    // this can be removed with a good way to detect the "Steve auth startup complete" log message
    await sleep(5*60)

    for (const count of [100, 400, 1600, 6400]) {
        for (const cluster of [upstream, downstream]) {
            k6_run(tester, { BASE_URL: `https://${cluster["private_name"]}:6443`, KUBECONFIG: cluster["kubeconfig"], CONTEXT: cluster["context"], COUNT: count}, {}, "k6/create_config_maps.js")
        }

        for (const test of ["load_steve_k8s_pagination", "load_steve_new_pagination"]) {
            for (const cluster of ["local", downstreamId]) {
                // warmup
                k6_run(tester,
                    {VUS: 1, PER_VU_ITERATIONS: 5, BASE_URL: privateRancherUrl, USERNAME: "admin", PASSWORD: ADMIN_PASSWORD, CLUSTER: cluster, CONFIG_MAP_COUNT: count },
                    {commit: commit, config_map_count: count, tag: tag, cluster: cluster, test: test},
                    `k6/${test}.js`)

                // test + record
                k6_run(tester,
                    {VUS: 10, PER_VU_ITERATIONS: 30, BASE_URL: privateRancherUrl, USERNAME: "admin", PASSWORD: ADMIN_PASSWORD, CLUSTER: cluster, CONFIG_MAP_COUNT: count },
                    {commit: commit, config_map_count: count, tag: tag, cluster: cluster, test: test},
                    `k6/${test}.js`,
                    true)
            }
        }
    }
}
