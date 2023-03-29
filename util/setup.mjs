#!/usr/bin/env node
import {ADMIN_PASSWORD, cd, run, runCollectingOutput} from "./common.mjs"

cd("terraform")

run("terraform", "init")
run("terraform", "apply", "-auto-approve")

const output = runCollectingOutput("terraform", "output", "-json")
const params = JSON.parse(output)
const baseUrl = params["base_url"]["value"]
const bootstrapPassword = params["bootstrap_password"]["value"]
const importedClusterNames = params["downstream_cluster_names"]["value"].join(",")

cd("k6")
run("k6", "run",
    "-e", `BASE_URL=${baseUrl}`,
    "-e", `BOOTSTRAP_PASSWORD=${bootstrapPassword}`,
    "-e", `PASSWORD=${ADMIN_PASSWORD}`,
    "-e", `IMPORTED_CLUSTER_NAMES=${importedClusterNames}`,
    "./rancher_setup.js"
)
