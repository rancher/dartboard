#!/usr/bin/env node
import {tofuDir, tofuVar, q, run} from "./lib/common.mjs"

run(`tofu -chdir=${q(tofuDir())} init -upgrade`)
run(`tofu -chdir=${q(tofuDir())} destroy -auto-approve ${q(tofuVar())}`)
