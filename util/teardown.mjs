#!/usr/bin/env node
import {cd, run} from "./common.mjs"

cd("terraform")

run("terraform", "init")

// HACK: Helm deployer does not always clean up successfully. Get rid of its state, cluster is being recreated anyway
const states = run("terraform", "state", "list").split("\n")
for (const i in states) {
    const state = states[i]
    if (state.indexOf("helm_release") > 0 && state !== ""){
        run("terraform", "state", "rm", state)
    }
}

run("terraform", "destroy", "-auto-approve")
