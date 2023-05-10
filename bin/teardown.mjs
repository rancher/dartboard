#!/usr/bin/env node
import {dir, run} from "./lib/common.mjs"

run(`terraform -chdir=${dir("terraform")} init`)
run(`terraform -chdir=${dir("terraform")} destroy -auto-approve`)
