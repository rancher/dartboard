#!/usr/bin/env node
import {ADMIN_PASSWORD, dir, run, runCollectingJSONOutput, runCollectingOutput} from "./lib/common.mjs"


run(`terraform -chdir=${dir("terraform")} init`)
run(`terraform -chdir=${dir("terraform")} apply -auto-approve`)

const params = runCollectingJSONOutput(`terraform -chdir=${dir("terraform")} output -json`)
const baseUrl = params["base_url"]["value"]
const bootstrapPassword = params["bootstrap_password"]["value"]
const upstreamCluster = params["upstream_cluster"]["value"]
const importedClusters = params["downstream_clusters"]["value"]
const importedClusterNames = importedClusters.map(c => c["name"])

run(`k6 run -e BASE_URL=${baseUrl} -e BOOTSTRAP_PASSWORD=${bootstrapPassword} -e PASSWORD=${ADMIN_PASSWORD} -e IMPORTED_CLUSTER_NAMES=${importedClusterNames} ${dir("k6")}/rancher_setup.js`)

const uka = `--kubeconfig=${upstreamCluster["kubeconfig"]}`
const uca = `--context=${upstreamCluster["context"]}`

// import clusters via curl | kubectl apply
for (const i in importedClusters) {
    const name = importedClusters[i]["name"]
    const dka = `--kubeconfig=${importedClusters[i]["kubeconfig"]}`
    const dca = `--context=${importedClusters[i]["context"]}`

    const clusterId = runCollectingJSONOutput(`kubectl get -n fleet-default cluster ${name} -o json ${uka} ${uca}`)["status"]["clusterName"]
    const token = runCollectingJSONOutput(`kubectl get -n ${clusterId} clusterregistrationtoken.management.cattle.io default-token -o json ${uka} ${uca}`)["status"]["token"]

    const url = `${baseUrl}/v3/import/${token}_${clusterId}.yaml`
    const yaml = runCollectingOutput(`curl --insecure -fL ${url}`)
    run(`kubectl apply -f - ${dka} ${dca}`, {input: yaml})
}

run(`kubectl wait clusters.management.cattle.io --all --for condition=ready=true --timeout=1h ${uka} ${uca}`)
run(`kubectl wait cluster.fleet.cattle.io --all --namespace fleet-default --for condition=ready=true --timeout=1h ${uka} ${uca}`)

console.log("\n")
console.log(`***Rancher UI:\n    ${baseUrl} (admin/${ADMIN_PASSWORD})`)
console.log("")
console.log(`***upstream cluster access:\n    ${uka} ${uca}`)

for (const i in importedClusters) {
    const name = importedClusters[i]["name"]
    const dka = `--kubeconfig=${importedClusters[i]["kubeconfig"]}`
    const dca = `--context=${importedClusters[i]["context"]}`
    console.log("")
    console.log(`***${name} cluster access:\n    ${dka} ${dca}`)
}
console.log("")
