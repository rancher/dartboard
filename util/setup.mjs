#!/usr/bin/env node
import {ADMIN_PASSWORD, cd, run, runCollectingJSONOutput, runCollectingOutput} from "./common.mjs"

cd("terraform")

run("terraform init")
run("terraform apply -auto-approve")

const params = runCollectingJSONOutput("terraform output -json")
const baseUrl = params["base_url"]["value"]
const bootstrapPassword = params["bootstrap_password"]["value"]
const upstreamCluster = params["upstream_cluster"]["value"]
const importedClusters = params["downstream_clusters"]["value"]
const importedClusterNames = importedClusters.map(c => c["name"])

cd("k6")
run(`k6 run -e BASE_URL=${baseUrl} -e BOOTSTRAP_PASSWORD=${bootstrapPassword} -e PASSWORD=${ADMIN_PASSWORD} -e IMPORTED_CLUSTER_NAMES=${importedClusterNames} ./rancher_setup.js`)

const uka = upstreamCluster["kubeconfig"] ? ` --kubeconfig=${upstreamCluster["kubeconfig"]}` : ""
const uca = upstreamCluster["context"] ? ` --context=${upstreamCluster["context"]}` : ""

for (const i in importedClusters) {
    const name = importedClusters[i]["name"]
    const dka = importedClusters[i]["kubeconfig"] ? ` --kubeconfig=${importedClusters[i]["kubeconfig"]}` : ""
    const dca = importedClusters[i]["context"] ? ` --context=${importedClusters[i]["context"]}` : ""

    const clusterId = runCollectingJSONOutput(`kubectl get -n fleet-default cluster ${name} -o json` + uka + uca)["status"]["clusterName"]
    const token = runCollectingJSONOutput(`kubectl get -n ${clusterId} clusterregistrationtoken.management.cattle.io default-token -o json` + uka + uca)["status"]["token"]

    const url = `${baseUrl}/v3/import/${token}_${clusterId}.yaml`
    const yaml = runCollectingOutput(`curl --insecure -fL ${url}`)
    run("kubectl apply -f -" + dka + dca, {input: yaml})
}

console.log("\n")
console.log(`***Rancher UI:\n    ${baseUrl} (admin/${ADMIN_PASSWORD})`)
console.log("")
console.log(`***upstream cluster access:\n   ${uka +uca}`)

for (const i in importedClusters) {
    const name = importedClusters[i]["name"]
    const dka = importedClusters[i]["kubeconfig"] ? ` --kubeconfig=${importedClusters[i]["kubeconfig"]}` : ""
    const dca = importedClusters[i]["context"] ? ` --context=${importedClusters[i]["context"]}` : ""
    console.log("")
    console.log(`***${name} cluster access:\n   ${dka + dca}`)
}
console.log("")
