#!/bin/bash

set -xe

cd terraform
terraform init
terraform apply -auto-approve

cd k6
k6 run -e BASE_URL=https://upstream.local.gd:8443 -e BOOTSTRAP_PASSWORD=admin -e PASSWORD=adminadminadmin ./rancher_setup.js
