#!/usr/bin/env node
import {run} from "./lib/common.mjs";

run("k3d image import --cluster moio-upstream rancher/rancher:baseline rancher/rancher:vai")
run("k3d image import --cluster moio-downstream rancher/rancher-agent:baseline rancher/rancher-agent:vai")
