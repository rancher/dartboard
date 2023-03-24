#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
cd $SCRIPT_DIR/..

set -xe

cd terraform

# HACK: helm provider does not always clean things up well. Just drop its stuff from state, we are deleting the cluster anyway
terraform state list | grep helm | xargs -n1 terraform state rm || true
terraform destroy -auto-approve
