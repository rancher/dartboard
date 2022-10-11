#!/bin/bash

set -xe

export KUBECONFIG=./config/upstream.yaml

nodes=`kubectl get node -o custom-columns=":metadata.name" --no-headers`

mkdir -p cypress/cypress/logs

for node in $nodes; do
  ./config/ssh-to-*-$node.sh "curl https://raw.githubusercontent.com/rancherlabs/support-tools/master/collection/rancher/v2.x/logs-collector/rancher2_logs_collector.sh | sudo bash -s"
  ./config/ssh-to-*-$node.sh 'cat $(ls /tmp/*.tar.gz | sort | tail -1)' >./cypress/cypress/logs/$node.tar.gz
done
