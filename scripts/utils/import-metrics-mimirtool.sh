#!/usr/bin/env bash

# set kubeconfig
export KUBECONFIG=$1

# run mimirtool, keep shell open
screen -t -s -d -m bash -c "kubectl -n cattle-monitoring-system run mimirtool --rm -ti --image=grafana/mimirtool:2.13.0 --env config.expand-env=true --env querier.max-samples=100000000 --command -- sh" &
