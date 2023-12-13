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


// Parameters
const CONFIG_MAP_COUNT = 10000
const SECRET_COUNT = 100
const ROLE_COUNT = 200
const USER_COUNT = 200
const PROJECT_COUNT = 50

// Refresh k6 files on the tester cluster
const clusters = runCollectingJSONOutput(`terraform -chdir=${terraformDir()} output -json`)["clusters"]["value"]
const tester = clusters["tester"]
helm_install("k6-files", dir("charts/k6-files"), tester, "tester", {})

const commit = runCollectingOutput("git rev-parse --short HEAD").trim()
const downstreams = Object.entries(clusters).filter(([k,v]) => k.startsWith("downstream"))
const upstream = clusters["upstream"]


// create users and roles
const upstreamAddresses = getAppAddressesFor(upstream)
const rancherURL = upstreamAddresses.localNetwork.httpsURL
const rancherClusterNetworkURL = upstreamAddresses.clusterNetwork.httpsURL
k6_run(tester,
    { BASE_URL: rancherClusterNetworkURL, USERNAME: "admin", PASSWORD: ADMIN_PASSWORD, ROLE_COUNT: ROLE_COUNT, USER_COUNT: USER_COUNT },
    {commit: commit, cluster: "upstream", test: "create_roles_users.mjs", Roles: ROLE_COUNT, Users: USER_COUNT},
    "k6/create_roles_users.js", false
)
// create projects
k6_run(tester,
    { BASE_URL: rancherClusterNetworkURL, USERNAME: "admin", PASSWORD: ADMIN_PASSWORD, PROJECT_COUNT: PROJECT_COUNT },
    {commit: commit, cluster: "upstream", test: "create_projects.mjs", Projects: PROJECT_COUNT},
    "k6/create_projects.js", false
)
// Create config maps
k6_run(tester,
    { BASE_URL: upstream["private_kubernetes_api_url"], KUBECONFIG: upstream["kubeconfig"], CONTEXT: upstream["context"], CONFIG_MAP_COUNT: CONFIG_MAP_COUNT, SECRET_COUNT: SECRET_COUNT},
    {commit: commit, cluster: "upstream", test: "create_load.mjs", ConfigMaps: CONFIG_MAP_COUNT, Secrets: SECRET_COUNT},
    "k6/create_k8s_resources.js", false
)
