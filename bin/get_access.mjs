#!/usr/bin/env node
import {
    ADMIN_PASSWORD,
    dir,
    terraformDir,
    helm_install,
    q,
    runCollectingJSONOutput,
    runCollectingOutput,
    getAppAddressesFor,
} from "./lib/common.mjs"
import {k6_run} from "./lib/k6.mjs"

const clusters = runCollectingJSONOutput(`terraform -chdir=${terraformDir()} output -json`)["clusters"]["value"]

const upstream = clusters["upstream"]
const upstreamAddresses = getAppAddressesFor(upstream)
const rancherURL = upstreamAddresses.localNetwork.httpsURL
const rancherClusterNetworkURL = upstreamAddresses.clusterNetwork.httpsURL

const downstreams = Object.entries(clusters).filter(([k,v]) => k.startsWith("downstream"))

const tester = clusters["tester"]
const testerAddresses = getAppAddressesFor(tester)

// Output access details
console.log("*** ACCESS DETAILS")
console.log()

console.log(`*** UPSTREAM CLUSTER`)
console.log(`    Rancher UI: ${rancherURL} (admin/${ADMIN_PASSWORD})`)

console.log(`    Kubernetes API:`)
console.log(`export KUBECONFIG=${q(upstream["kubeconfig"])}`)
console.log(`kubectl config use-context ${q(upstream["context"])}`)
for (const [node, command] of Object.entries(upstream["node_access_commands"])) {
    console.log(`    Node ${node}: ${q(command)}`)
}
console.log()

for (const [name, downstream] of downstreams) {
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
const grafanaURL = testerAddresses.localNetwork.httpURL
console.log(`    Grafana UI: ${grafanaURL}/grafana/d/a1508c35-b2e6-47f4-94ab-fec400d1c243/test-results?orgId=1&refresh=5s&from=now-30m&to=now (admin/${ADMIN_PASSWORD})`)
console.log(`    Kubernetes API:`)
console.log(`export KUBECONFIG=${q(tester["kubeconfig"])}`)
console.log(`kubectl config use-context ${q(tester["context"])}`)
for (const [node, command] of Object.entries(tester["node_access_commands"])) {
    console.log(`    Node ${node}: ${q(command)}`)
}
console.log(`    Interactive k6 benchmark run:`)
console.log(`kubectl run -it --rm k6-manual-run --image=grafana/k6:latest --command sh`)
console.log(`k6 run -e BASE_URL=${rancherClusterNetworkURL} -e USERNAME=admin -e PASSWORD=adminadminadmin ./steve_paginated_api_benchmark.js`)
console.log()
