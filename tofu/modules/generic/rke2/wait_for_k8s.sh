#!/usr/bin/env bash

set -xe

source /root/.bash_profile

WAITSECS=${WAITSECS:-"2"}
MAX_ATTEMPTS=${MAX_ATTEMPTS:-"20"}
KUBECONFIG=${KUBECONFIG:-"/etc/rancher/rke2/rke2.yaml"}

for ((i = 1; i <= MAX_ATTEMPTS; i++))
do
  if ! systemctl is-active --quiet rke2-server && ! systemctl is-active --quiet rke2-agent; then
    echo "rke2 service is not active yet..."
    sleep "$WAITSECS"
    continue
  fi

  if [ ! -s "$KUBECONFIG" ]; then
    echo "Kubeconfig not available yet at ${KUBECONFIG}..."
    sleep "$WAITSECS"
    continue
  fi

  if kubectl get --raw=/readyz >/dev/null 2>&1
  then
    break
  fi

  echo "Waiting another ${WAITSECS} seconds for k8s API to be up..."
  sleep "$WAITSECS"
done

if [ $i -gt $MAX_ATTEMPTS ]; then
  echo "ERROR: k8s API still not up after ${MAX_ATTEMPTS} attempts, quitting"
  systemctl status rke2-server rke2-agent --no-pager -l || true
  journalctl -u rke2-server -u rke2-agent --no-pager -n 200 || true
  exit 1
fi

echo "Waiting for the RKE2 ingress controller to be up..."
kubectl rollout status daemonset \
  rke2-ingress-nginx-controller \
  -n kube-system \
  --timeout 600s
