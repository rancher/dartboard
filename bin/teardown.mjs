#!/usr/bin/env node
import {terraformDir, terraformVar, q, run} from "./lib/common.mjs"

run(`tofu -chdir=${q(terraformDir())} init -upgrade`)
run(`tofu -chdir=${q(terraformDir())} destroy -auto-approve ${q(terraformVar())}`)
