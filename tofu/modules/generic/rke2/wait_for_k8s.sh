#!/usr/bin/env bash

set -xe

source /root/.bash_profile

while ! kubectl get services
do
  echo "Waiting for k8s API to be up..."
  sleep 3
done

echo "Waiting for the RKE2 ingress controller to be up..."
kubectl rollout status daemonset \
  rke2-ingress-nginx-controller \
  -n kube-system \
  --timeout 600s
