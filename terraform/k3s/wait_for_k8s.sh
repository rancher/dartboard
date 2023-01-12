#!/usr/bin/env bash

set -xe

for i in {1..20}
do
  if kubectl get services
  then
    exit 0
  fi
  echo "Waiting for k8s API to be up..."
  sleep 2
done

echo "ERROR: k8s API still not up after 20 attempts, quitting"
exit 1
