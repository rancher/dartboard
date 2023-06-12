#!/usr/bin/env node
import {terraformDir, q, run} from "./lib/common.mjs"

run(`terraform -chdir=${q(terraformDir())} init`)
run(`terraform -chdir=${q(terraformDir())} destroy -auto-approve`)
