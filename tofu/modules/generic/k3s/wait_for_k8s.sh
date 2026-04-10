#!/usr/bin/env bash

set -xe

WAITSECS=${WAITSECS:-"2"}
MAX_ATTEMPTS=${MAX_ATTEMPTS:-"20"}
KUBECTL=${KUBECTL:-"/usr/local/bin/kubectl"}
K3S=${K3S:-"/usr/local/bin/k3s"}
KUBECONFIG=${KUBECONFIG:-"/etc/rancher/k3s/k3s.yaml"}

for ((i = 1; i <= MAX_ATTEMPTS; i++))
do
  if ! systemctl is-active --quiet k3s; then
    echo "k3s service is not active yet..."
    sleep "$WAITSECS"
    continue
  fi

  if [ ! -s "$KUBECONFIG" ]; then
    echo "Kubeconfig not available yet at ${KUBECONFIG}..."
    sleep "$WAITSECS"
    continue
  fi

  if [ -x "$K3S" ] && "$K3S" kubectl --kubeconfig "$KUBECONFIG" get --raw=/readyz >/dev/null 2>&1
  then
    exit 0
  fi

  if "$KUBECTL" --kubeconfig "$KUBECONFIG" get --raw=/readyz >/dev/null 2>&1
  then
    exit 0
  fi

  echo "Waiting another ${WAITSECS} seconds for k8s API to be up..."
  sleep "$WAITSECS"
done

echo "ERROR: k8s API still not up after ${MAX_ATTEMPTS} attempts, quitting"
systemctl status k3s --no-pager -l || true
journalctl -u k3s --no-pager -n 200 || true
exit 1
