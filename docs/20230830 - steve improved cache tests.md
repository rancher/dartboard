# 2023-08-30 - Steve improved cache tests

## Results

TBD.

## Requirements

Development environment:
- a Docker host with at least 16 GiB of available RAM

AWS test environment:
- API access to EC2 configured for your terminal
  - for SUSE Engineering:
    - [have "AWS Landing Zone" added to your Okta account](https://confluence.suse.com/display/CCOE/Requesting+AWS+Access)
    - open [Okta](https://suse.okta.com/) -> "AWS Landing Zone"
    - Click on "AWS Account" -> your account -> "Command line or programmatic access" -> click to copy commands under "Option 1: Set AWS environment variables"
    - paste contents in terminal

All environments:
- `kubectl`
- `helm`
- [Terraform](https://www.terraform.io/)
- `git`
- `node`

## Hardware and infrastructure configuration outline

Development environment:
- three k3d clusters (an upstream, a downstream, a "tester" cluster generating load and collecting metrics)
  - each cluster with 1 node, etcd backed storage forced
  - backing hardware: one Lenovo ThinkPad P51 laptop (circa 2017)
    - Intel(R) Core(TM) i7-7820HQ CPU @ 2.90GHz, 8 vCPUs
    - 32 GiB RAM
    - 2x SSD local storage (mdraid 0, xfs)

AWS test environment TBD.

## Architecture

The test is conducted on 3 clusters:
- one upstream cluster running Rancher. It also has Rancher Monitoring installed and configured to forward relevant metrics to the tester cluster
- one tester cluster running Mimir (for long-term metric storage) and Grafana (to consult data collected in Mimir)
- one downstream cluster imported into Rancher

VM/Docker container creation and Kubernetes installation is performed via [Terraform](https://www.terraform.io/). Modules implement both options with compatible interfaces, so they are easily exchangeable.

All Kubernetes applications including Rancher are installed and configured by Node scripts, using `helm` and `kubectl` to perform cluster operations.

All load testing is implemented in [k6](https://k6.io/). k6 runs from inside the tester cluster and is configured to forward testing metrics to Mimir as well. k6 scripts have facilities to access the Kubernetes API as well as the Rancher API of upstream and downstream clusters.


## Full configuration details

- All configuration is stored in this repo in the [20230830_steve_improved_cache_tests](https://github.com/moio/scalability-tests/tree/20230830_steve_improved_cache_tests/) branch. All references below are relative to that branch
- All infrastructure is defined in Terraform files in the [terraform](../terraform) directory, branch of the directory
- Kubernetes application installation and configuration is defined in Javascript files in the [bin](../bin) directory
- Ad-hoc charts are defined in the [charts](../charts) directory
- Load test scripts and configuration are defined in Javascript files in the [k6](../k6) directory

It is expected that all scripts and their parts are idempotent.

It is expected that tests are fully reproducible given the same commit in this repo.


## Reproduction Instructions

Build patched images:
```shell
# set up build
git clone https://github.com/rmweir/rancher.git
cd rancher

cat >scripts/quickbuild <<"EOF"
#!/bin/bash
set -e

cd $(dirname $0)

./build
./package
EOF
chmod +x scripts/quickbuild

# build Rancher from the [rmweir/improved](https://github.com/rmweir/rancher/tree/improved) branch
git checkout improved
TAG=improved make quickbuild
```

Deploy the k3d infrastructure, install Rancher, set up clusters for tests, import built images:
```shell
./bin/setup.mjs && ./bin/import_images.mjs
```


All created infrastructure can be destroyed at the end of the test via:
```shell
./teardown.mjs
```

### Run tests

#### Outline
First, we create a given number of ConfigMaps in a test namespace via a k6 script. Each ConfigMap is created with 10 kb of data payload.

Then, we simulate 10 virtual users listing all ConfigMaps in that namespace via another k6 script. Each user will repeat the listing 30 times (for statistical accuracy of measures). The page size is of 100, like in the current UI. We exercise both the k8s based pagination implementation, using the `limit`/`continue` parameters and currently used by the [dashboard](https://github.com/rancher/dashboard/) UI, as well as the new Steve-cache pagination implementation using the `page`/`pagesize` parameters. We test both local and downstream cluster. Tests are repeated for `baseline` and `improved` images.

Details on tests are available in the [bin/run_test.js](../bin/run_tests.mjs) script source file.

#### Procedure

```shell
./run_tests.mjs
```
