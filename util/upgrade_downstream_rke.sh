#!/bin/bash

set -xe

export KUBECONFIG=./config/downstream.yaml

server_nodes=`kubectl get node --selector='node-role.kubernetes.io/master' -o custom-columns=":metadata.name" --no-headers`

for node in $server_nodes; do
  kubectl drain --delete-emptydir-data --ignore-daemonsets $node
  ./config/ssh-to-*-$node.sh "sh -c 'curl -sfL https://get.rke2.io | INSTALL_RKE2_VERSION=v1.23.10+rke2r1 sh -; sudo systemctl restart rke2-server'"
  kubectl uncordon $node
  sleep 10
done

agent_nodes=`kubectl get node --selector='!node-role.kubernetes.io/master' -o custom-columns=":metadata.name" --no-headers`

for node in $agent_nodes; do
  kubectl drain --delete-emptydir-data --ignore-daemonsets $node
  ./config/ssh-to-*-$node.sh "sh -c 'curl -sfL https://get.rke2.io | INSTALL_RKE2_VERSION=v1.23.10+rke2r1 sh -; sudo systemctl restart rke2-agent'"
  kubectl uncordon $node
  sleep 10
done
