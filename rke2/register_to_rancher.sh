#!/usr/bin/env bash
set -xe

curl --insecure --location ${manifest_url} | \
  kubectl apply --server @{server_url} \
    --client-certificate /var/lib/rancher/rke2/server/tls/client-ca.crt \
    --client-key /var/lib/rancher/rke2/server/tls/client-ca.key \
    --certificate-authority /var/lib/rancher/rke2/server/tls/server-ca.crt -f -
