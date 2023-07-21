#!/usr/bin/env bash

set -xe

WAITSECS=${WAITSECS:-"2"}
KUBECTL=${KUBECTL:-"/usr/local/bin/kubectl"}

for i in {1..20}
do
  if $KUBECTL get services
  then
    exit 0
  fi
  echo "Waiting another ${WAITSECS} seconds for k8s API to be up..."
  sleep $WAITSECS
done

echo "ERROR: k8s API still not up after 20 attempts, quitting"
exit 1
