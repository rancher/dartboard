# 2022-11-30 - can-i API microbenchmark

## Results

A microbenchmark on the "can-i" Kubernetes API endpoint ([LocalSubjectAccessReview](https://kubernetes.io/docs/reference/access-authn-authz/authorization/)) shows it returns in milliseconds from the Python Kubernetes client library.

Exact result output is:
```
repetitions: 1000, mean runtime (s): 0.004, stdev (s): 0.001, stdev (%): 0.22
```

## Methodology notes

- Rancher is installed on a 3-node "medium sized" k3s cluster backed on embedded etcd
- a benchmark script is run to determine performance of the k8s API, specifically the response time
- no workload is running on the local cluster, no downstream clusters are registered
- measurement is repeated 1000 times and average/standard deviation is calculated. All standard deviations observed appeared to be reasonable to sustain conclusions above

## Test outline
- infrastructure is set up:
    - AWS hardware (VMs, network devices, databases...) are deployed
    - k3s is installed on cluster nodes, Rancher is installed on top of it
- test is conducted:
    - initial admin user is set up ([detail](../cypress/cypress/e2e/users.cy.js))
    - the benchmark script is run (see below)

## AWS Hardware configuration outline

- bastion host (for SSH tunnelling only): `t4g.small`, 50 GiB EBS `gp3` root volume
- Rancher cluster: 3-node `t3a.xlarge` (4 vCPU, 16 GiB RAM), 50 GiB EBS `gp3` root volume
- networking: one /16 AWS VPC with two /24 subnets
    - public subnet: contains the one bastion host which exposes port 22 to the Internet via security groups
    - private subnet: contains all other nodes. Traffic allowed only internally and to/from the bastion via SSH

References:
- [instance types](https://aws.amazon.com/ec2/instance-types/)
- [EBS](https://aws.amazon.com/ebs/)
- [VPC](https://aws.amazon.com/vpc/)

## Software configuration outline

- bastion host: SLES 15 SP4
- k3s cluster: Rancher 2.6.9 on a 3-node v1.24.6+k3s1 cluster
    - all nodes based on Ubuntu Jammy 22.04 LTS amd64

## Full configuration details

All infrastructure is defined via [Terraform](https://www.terraform.io/) files in the [20221130_can-i_microbenchmark](https://github.com/rancher/dartboard/tree/20221128_api_load_benchmarks/terraform) branch. Note in particular [inputs.tf](https://github.com/rancher/dartboard/blob/20221130_can-i_microbenchmark/terraform/inputs.tf) for the main parameters.
Initial configuration is driven by [Cypress](https://www.cypress.io/) files in the [cypress/e2e](https://github.com/rancher/dartboard/tree/20221130_can-i_microbenchmark/cypress/cypress/e2e) directory.
Benchmark Python scripts are available in the [util](https://github.com/rancher/dartboard/tree/20221130_can-i_microbenchmark/util) directory.

## Reproduction Instructions

### Requirements

- API access to EC2 configured for your terminal
    - for SUSE Engineering:
        - [have "AWS Landing Zone" added to your Okta account](https://confluence.suse.com/display/CCOE/Requesting+AWS+Access)
        - open [Okta](https://suse.okta.com/) -> "AWS Landing Zone"
        - Click on "AWS Account" -> your account -> "Command line or programmatic access" -> click to copy commands under "Option 1: Set AWS environment variables"
        - paste contents in terminal
- [Terraform](https://www.terraform.io/downloads)
- `git`
- `node`
- `nc` (netcat)

### Setup

- clone this project:
```shell
git clone https://github.com/rancher/dartboard.git
cd scalability-tests
git checkout 20221130_can-i_microbenchmark
```
- initialize Terraform and Cypress:
```shell
cd terraform
terraform init
cd ../cypress
npm install cypress --save-dev
```

### Run

Configure and deploy the AWS infrastructure:
- edit `terraform/inputs.tf` (specifically: `ssh_private_key_path`, `ssh_public_key_path`)
- deploy and configure infrastructure:
```shell
./util/deploy_benchmark_infrastructure.sh
```

Note that the `deploy_benchmark_infrastructure.sh` is idempotent, it will destroy and re-create the cluster if run multiple times. This makes it easier to repeat tests (possibly with different configuration).


Execute the benchmark:
```shell
./config/ssh-to-upstream-server-node-0-*.sh KUBECONFIG=/etc/rancher/k3s/k3s.yaml python3 - <./util/benchmark_cani.py | tee -a results.csv
```

Elements of the line above have the following meaning:
- `./config/ssh-to-upstream-server-node-0-*.sh` opens an SSH shell to the first server node of the cluster
- `KUBECONFIG` points to the configuration file on the first cluster node
- `./util/benchmark_cani.py` is the benchmark script - other available options are `./util/benchmark_k8s_config_maps.py` and `./util/benchmark_steve_secrets.py`, all accept the same parameters above
- `| tee -a results.csv` saves results into a file that can be opened in a spreadsheet editor

### Cleanup

All created infrastructure can be destroyed via:
```shell
terraform destroy -auto-approve
```
