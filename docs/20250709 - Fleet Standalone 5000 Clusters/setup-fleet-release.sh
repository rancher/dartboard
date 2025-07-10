#!/bin/bash

set -euxo pipefail

./default_config/open-tunnels-to-upstream-server-0.sh
export KUBECONFIG="$PWD/default_config/upstream.yaml"

kubectl create ns fleet-default

helm -n cattle-fleet-system upgrade --install --create-namespace --wait fleet-crd \
  https://github.com/rancher/fleet/releases/download/v0.13.0-beta.2/fleet-crd-0.13.0-beta.2.tgz

node=$(kubectl get nodes --selector='!node-role.kubernetes.io/control-plane' -o jsonpath="{.items[*].status.addresses[?(@.type=='Hostname')].address}" | tr ' ' '\n' | shuf -n 1)
kubectl label node "$node" role=agent

cpnode=$(kubectl get nodes --selector='node-role.kubernetes.io/control-plane' -o jsonpath="{.items[*].status.addresses[?(@.type=='Hostname')].address}" | tr ' ' '\n' | head -1 )

ca=$( kubectl config view --flatten -o jsonpath='{.clusters[?(@.name == "upstream")].cluster.certificate-authority-data}' | base64 -d )
helm -n cattle-fleet-system upgrade --install --create-namespace --wait \
  --set bootstrap.enabled=false \
  --set controller.reconciler.workers.gitrepo=100 \
  --set controller.reconciler.workers.bundle=200 \
  --set controller.reconciler.workers.bundledeployment=200 \
  --set controller.reconciler.workers.cluster=100 \
  --set apiServerCA="$ca" \
  --set apiServerURL="https://$cpnode.ec2.internal:6443" \
  --set nodeSelector.role=agent \
  fleet \
  https://github.com/rancher/fleet/releases/download/v0.13.0-beta.2/fleet-0.13.0-beta.2.tgz

kubectl -n cattle-fleet-system rollout status deploy/fleet-controller
