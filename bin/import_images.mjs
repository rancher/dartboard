#!/usr/bin/env node
import {run} from "./lib/common.mjs";

run("k3d image import --cluster st-upstream rancher/rancher:improved")
run("k3d image import --cluster st-downstream rancher/rancher-agent:improved")
