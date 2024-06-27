#!/usr/bin/env bash

set -xe

source /root/.bash_profile

while ! kubectl get services
do
  echo "Waiting for k8s API to be up..."
  sleep 3
done
