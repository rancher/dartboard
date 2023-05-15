#!/usr/bin/env node
import {dir, q, run} from "./lib/common.mjs"

run(`terraform -chdir=${q(dir("terraform"))} init`)
run(`terraform -chdir=${q(dir("terraform"))} destroy -auto-approve`)
