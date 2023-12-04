# 2023-12-01 - Comparison between AKS and RKE2

## Results

Under test conditions, according to collected measures described below:

TBD

## Hardware and infrastructure configuration outline

AKS test environment: 
 - upstream AKS cluster, Standard SKU tier, 3 `Standard_D8ds_v4` worker nodes (8 vCPU, 32 GB RAM, 300 GB ephemeral SSD) plus one worker node dedicated to monitoring exclusively
 - 2 downstream clusters, k3s, 1 `Standard_B1ms` node (1 vCPU, 2 GB RAM, 4 GB ephemeral SSD)
 - 1 tester cluster, k3s, 1 `Standard_B2as_v2` node (2 vCPU, 4 GB RAM, 8 GB ephemeral SSD)
 - all VMs run in an isolated network. Access happens primarily via SSH tunnels. The upstream cluster Kubernetes API and Rancher UI are also exposed to the Internet but not exercised during tests

RKE2 test environment:
  - upstream RKE2 cluster, 3 Standard_D8ds_v4 server nodes (8 vCPU, 32 GB RAM, 300 GB ephemeral SSD) plus one worker node dedicated to monitoring exclusively
  - 2 downstream clusters, k3s, 1 Standard_B1ms node (1 vCPU, 2 GB RAM, 4 GB ephemeral SSD)
  - 1 tester cluster, k3s, 1 `Standard_B2as_v2` node (2 vCPU, 4 GB RAM, 8 GB ephemeral SSD)
  - all VMs run in an isolated network. Access happens exclusively via SSH tunnels

## Process outline

- for each environment:
  - infrastructure setup:
    - upstream cluster is deployed, Rancher is installed and configured
    - downstream clusters are deployed and imported into Rancher
    - tester cluster is deployed, Mimir and Grafana are installed and configured
  - test execution:
    - k6 load test scripts are run in the tester cluter to exercise the Rancher API (in the upstream cluster)

## Full configuration details

All infrastructure is defined in [Terraform](https://www.terraform.io/) files in the [20231201_aks_rke_comparison](https://github.com/moio/scalability-tests/tree/20231201_aks_rke_comparison/terraform) branch.

[k6](https://k6.io) load test scripts are defined in the [k6](https://github.com/moio/scalability-tests/tree/20231201_aks_rke_comparison/k6) directory.

## Reproduction Instructions

### Requirements

- [Terraform](https://www.terraform.io/downloads)
- `git`
- `nc` (netcat)
- `make`
- [k6](https://k6.io)
- `node`
- `azure-cli`: (on macOS, use [Homebrew](https://brew.sh/): `brew install azure-cli`)

### Setup

Log into Azure via the CLI:
  - for SUSE employees: log into [OKTA](https://suse.okta.com), click on "Azure Landing Zone Office Portal", run `az login`

Deploy the AKS environment, install Rancher, set up clusters for tests:
```shell
# clone this project
git clone -b 20231201_aks_rke_comparison https://github.com/moio/scalability-tests.git
cd scalability-tests

export TERRAFORM_WORK_DIR=terraform/main/aks

./bin/setup.mjs && ./bin/run_tests.mjs
````

Deploy the AKS environment, install Rancher, set up clusters for tests (in a different terminal):
```shell
cd scalability-tests

export TERRAFORM_WORK_DIR=terraform/main/azure

./bin/setup.mjs && ./bin/run_tests.mjs
```

All created infrastructure can be destroyed at the end of the test via:
```shell
./teardown.mjs
```

### Run tests

```shell
kubectl run -it --rm k6-manual-run --image=grafana/k6:latest --command sh
k6 run -e BASE_URL=${rancherClusterNetworkURL} -e USERNAME=admin -e PASSWORD=adminadminadmin ./steve_paginated_api_benchmark.js
```

Interpreting results: important output data points are:
 * `✓ checks`: number of successful checks. Presence of any errors invalidates the test
 * `http_req_duration`: duration of http requests to retrieve a page up to 100 resources
   * `avg` average duratio nof such requests
   * `min` minimum duration of such requests
   * `med` median duration of such requests
   * `max` maximum duration of such requests
   * `p(95)` 95th percentile - 95% of requests had a duration less than or equal to this value
   * `p(99)` 99th percentile - 99% of requests had a duration less than or equal to this value

## Follow-up notes
