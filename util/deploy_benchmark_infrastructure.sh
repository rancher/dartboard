#!/bin/bash

set -xe

pushd terraform
terraform state list | grep rancher | xargs -n1 terraform state rm
terraform destroy -target=module.upstream_cluster -auto-approve
terraform apply -auto-approve
popd
pushd cypress
./node_modules/cypress/bin/cypress run --spec ./cypress/e2e/users.cy.js
popd
