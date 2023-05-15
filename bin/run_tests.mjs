#!/usr/bin/env node
import {appendFileSync, readFileSync} from "fs";
import {ADMIN_PASSWORD, dir, q, run, runCollectingJSONOutput, sleep} from "./lib/common.mjs"

const params = runCollectingJSONOutput(`terraform -chdir=${dir("terraform")} output -json`)
const baseUrl = params["base_url"]["value"]

const upstreamCluster = params["upstream_cluster"]["value"]
const upstreamContext = upstreamCluster["context"]
const upstreamKubeconfig = upstreamCluster["kubeconfig"]
const downstreamCluster = params["downstream_clusters"]["value"][0]
const downstreamClusterId = runCollectingJSONOutput(`kubectl get -o json --kubeconfig=${q(upstreamKubeconfig)} --context=${q(upstreamContext)} -n fleet-default cluster ${downstreamCluster["name"]}`)["status"]["clusterName"]
const downstreamContext = downstreamCluster["context"]
const downstreamKubeconfig = downstreamCluster["kubeconfig"]

const stats = ['avg','min','med','max','p(95)','p(99)','count']

writeResultFileHeaders()
for (const tag of ["baseline", "vai"]) {
    run(`kubectl set image -n cattle-system deployment/rancher rancher=rancher/rancher:${q(tag)} --kubeconfig=${q(upstreamKubeconfig)} --context=${q(upstreamContext)}`)
    run(`kubectl rollout status --watch --timeout=3600s -n cattle-system deployment/rancher --kubeconfig=${q(upstreamKubeconfig)} --context=${q(upstreamContext)}`)
    run(`kubectl set image -n cattle-system deployment/cattle-cluster-agent cluster-register=rancher/rancher-agent:${q(tag)} --kubeconfig=${q(downstreamKubeconfig)} --context=${q(downstreamContext)}`)
    run(`kubectl rollout status --watch --timeout=3600s -n cattle-system deployment/cattle-cluster-agent --kubeconfig=${q(downstreamKubeconfig)} --context=${q(downstreamContext)}`)

    // HACK: allow 5 more minutes for Steve to start up on the remote cluster
    // this can be removed with a good way to detect the "Steve auth startup complete" log message
    await sleep(5*60)

    for (const count of [100, 400, 1600, 6400]) {
        for (const cluster of [upstreamCluster, downstreamCluster]) {
            run(`k6 run -e KUBECONFIG=${q(cluster["kubeconfig"])} -e CONTEXT=${q(cluster["context"])} -e COUNT=${count} ${q(dir("k6/create_config_maps.js"))}`)
        }

        for (const test of ["load_steve_k8s_pagination", "load_steve_new_pagination"]) {
            for (const cluster of ["local", downstreamClusterId]) {
                // warmup
                run(`k6 run -e VUS=1 -e PER_VU_ITERATIONS=5 -e BASEURL=${q(baseUrl)} -e USERNAME=admin -e PASSWORD=${q(ADMIN_PASSWORD)} -e CLUSTER=${q(cluster)} ${q(dir(`k6/${test}.js`))}`)

                // test + record
                run(`k6 run -e VUS=10 -e PER_VU_ITERATIONS=30 -e BASEURL=${q(baseUrl)} -e USERNAME=admin -e PASSWORD=${q(ADMIN_PASSWORD)} -e CLUSTER=${q(cluster)} --summary-trend-stats=${q(stats)} --summary-time-unit=ms ${q(dir(`k6/${test}.js`))}`)
                writeResultFileLine(tag, count, test, cluster)
            }
        }
    }
}

function writeResultFileHeaders() {
    const headers = `test run on ${new Date().toISOString()}\n` +
        `tag,count,test,cluster,${stats.join(',')}\n`
    appendFileSync(`results.csv`, headers)
}

function writeResultFileLine(tag, count, test, cluster) {
    const result = JSON.parse(readFileSync("out.json"))["metrics"]["http_req_duration"]["values"]
    appendFileSync(`results.csv`, `${tag},${count},${test},${cluster},${stats.map(s => result[s]).join(',')}\n`)
}
