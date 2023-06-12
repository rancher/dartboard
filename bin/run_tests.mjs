#!/usr/bin/env node
import {
    ADMIN_PASSWORD,
    dir,
    terraformDir,
    helm_install,
    q,
    runCollectingJSONOutput, runCollectingOutput,
} from "./lib/common.mjs"
import {k6_run} from "./lib/k6.mjs"


// Parameters
const CONFIG_MAP_COUNT = 1000
const SECRET_COUNT = 1000

// Refresh k6 files on the tester cluster
const clusters = runCollectingJSONOutput(`terraform -chdir=${terraformDir()} output -json`)["clusters"]["value"]
const tester = clusters["tester"]
helm_install("k6-files", dir("charts/k6-files"), tester, "tester", {})

// Create config maps
const commit = runCollectingOutput("git rev-parse --short HEAD").trim()
const downstreams = Object.entries(clusters).filter(([k,v]) => k.startsWith("downstream"))
for (const [name, downstream] of downstreams) {
    k6_run(tester,
        { BASE_URL: `https://${downstream["private_name"]}:6443`, KUBECONFIG: downstream["kubeconfig"], CONTEXT: downstream["context"], CONFIG_MAP_COUNT: CONFIG_MAP_COUNT, SECRET_COUNT: SECRET_COUNT},
        {commit: commit, cluster: name, test: "create_load.mjs", ConfigMaps: CONFIG_MAP_COUNT, Secrets: SECRET_COUNT},
        "k6/create_resources.js", true
    )
}

// Output access details
console.log("*** ACCESS DETAILS")
console.log()

const upstream = clusters["upstream"]
const upstreamSAN = upstream["san"]
const upstreamPublicPort = upstream["public_https_port"]
console.log(`*** UPSTREAM CLUSTER`)
console.log(`    Rancher UI: https://${upstreamSAN}:${upstreamPublicPort} (admin/${ADMIN_PASSWORD})`)
console.log(`    Kubernetes API:`)
console.log(`export KUBECONFIG=${q(upstream["kubeconfig"])}`)
console.log(`kubectl config set-context ${q(upstream["context"])}`)
console.log()

for (const [name, downstream] of downstreams) {
    console.log(`*** ${name.toUpperCase()} CLUSTER`)
    console.log(`    Kubernetes API:`)
    console.log(`export KUBECONFIG=${q(downstream["kubeconfig"])}`)
    console.log(`kubectl config set-context ${q(downstream["context"])}`)
    console.log()
}

console.log(`*** TESTER CLUSTER`)
const testerSAN = tester["san"]
const testerPublicPort = tester["public_http_port"]
console.log(`    Grafana UI: http://${testerSAN}:${testerPublicPort}/grafana/ (admin/${ADMIN_PASSWORD})`)
console.log()
