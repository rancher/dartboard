# 2023-03-24 - Constrained hardware tests

## Results

TBD. 

First objective of this research is to determine "background load" of idle Rancher with a defined configuration (number of clusters, nodes, workloads, etc.)

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
  - upstream cluster with 3 nodes (forcing etcd-based storage) and 2 agent nodes (one of which limited to run rancher-monitoring exclusively)
  - downstream and tester clusters single node
  - backing hardware: one Lenovo ThinkPad P51 laptop (circa 2017)
    - Intel(R) Core(TM) i7-7820HQ CPU @ 2.90GHz, 8 vCPUs
    - 32 GiB RAM
    - 2x SSD local storage (mdraid 0, xfs)

AWS test environment:
- three k3s clusters (an upstream, a downstream, a "tester" cluster generating load and collecting metrics)
  - upstream cluster with 3 nodes (forcing etcd-based storage) and 2 agent nodes (one of which limited to run rancher-monitoring exclusively)
    - all nodes i3.large running openSUSE 15.4, one 475 GiB NVMe SSD, 50 GiB network storage
  - downstream clusters single node, t4g.large, 50 GiB network storage
  - tester clusters single node, t3a.large, 50 GiB network storage
- see https://aws.amazon.com/ec2/instance-types/ for details

## Architecture

The test is conducted on 2+N clusters:
 - one upstream cluster running Rancher. It also has Rancher Monitoring installed and configured to forward relevant metrics to the tester cluster
 - one tester cluster running Mimir (for long-term metric storage) and Grafana (to consult data collected in Mimir)
 - N downstream clusters imported into Rancher

VM/Docker container creation and Kubernetes installation is performed via [Terraform](https://www.terraform.io/). Modules implement both options with compatible interfaces, so they are easily exchangeable.

All Kubernetes applications including Rancher are installed and configured by Node scripts, using `helm` and `kubectl` to perform cluster operations.

All load testing is implemented in [k6](https://k6.io/). k6 runs from inside the tester cluster and is configured to forward testing metrics to Mimir as well. k6 scripts have facilities to access the Kubernetes API as well as the Rancher API of upstream and downstream clusters.

## Full configuration details

- All infrastructure is defined in Terraform files in the [terraform](../terraform) directory
- Kubernetes application installation and configuration is defined in Javascript files in the [bin](../bin) directory
- Ad-hoc charts are defined in the [charts](../charts) directory
- Load test scripts and configuration are defined in Javascript files in the [k6](../k6) directory

It is expected that all scripts and their parts are idempotent.

It is expected that tests are fully reproducible given the same commit in this repo.

## Load outline

- all downstream clusters run Rancher Monitoring
- 1000 1kiB ConfigMaps per downstream cluster
- 1000 1kiB Secrets per downstream cluster
- more TBD

## Reproduction Instructions

```shell
export TERRAFORM_MAIN_DIR=aws # or k3d, by default
./setup.mjs && ./run_tests.mjs
```

Notes:
 - `./setup.mjs` invokes Terraform to deploy infrastructure and then Helm/kubectl for the configuration of applications
 - `./run_tests.mjs` invokes k6 to set up resources which will generate background load

All created infrastructure can be destroyed at the end of the test via:
```shell
./teardown.mjs
```

## Interpreting results

The script will output URLs to reach to the Mimir-backed Grafana dashboard.

Further interpretation instructions TBD
