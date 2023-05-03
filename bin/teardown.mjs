#!/usr/bin/env node
import {dir, run, runCollectingOutput} from "./lib/common.mjs"

run(`terraform -chdir=${dir("terraform")} init`)

// HACK: Helm deployer does not always clean up successfully. Get rid of its state, cluster is being destroyed anyway
const states = runCollectingOutput(`terraform -chdir=${dir("terraform")} state list`).split("\n")
for (const i in states) {
    const state = states[i]
    if (state.indexOf("helm_release") > 0 && state !== ""){
        run(`terraform -chdir=${dir("terraform")} state rm ${state}`)
    }
}

run(`terraform -chdir=${dir("terraform")} destroy -auto-approve`)
