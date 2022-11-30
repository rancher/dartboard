# 2022-11-28 - API load benchmarks

## Results

## Results

- Rancher is installed onto various 3-node "medium sized" k3s and RKE2 clusters, with different data stores (etcd, mariadb and postgres)
- 1k to 256k resources (secrets or configmaps, big or small data payload sizes) are created on them
- benchmarks are run to determine performance of the k8s API and Rancher's Steve to retrieve all created resources, specifically:
  - the response time
  - the amount of bytes transferred

Following conclusions are taken from result data:
 - Steve byte overhead seems to be ~550 bytes/secret compared to the pure k8s API
 - Transferred data is in the hundreds of megabytes at ~10k large (10kB) resources or ~100k small (4-byte) resources
 - Steve takes about ~7x the time the pure k8s API to return a response
 - Pure k8s API response times are in the seconds at >=~100k small resources or >=~20k big ones 
 - Steve response times are in the seconds at >=~15k small resources or >=~3k big ones
 - Configmaps take more processing time than similarly-sized secrets on k3s (up to 60%, usually less). Time is anyway in same order of magnitude
 - Configmaps and secrets take a comparable amount of time on RKE2
 - k3s is consistently faster than RKE2 at high scale and large resources (by ~15% up to ~60%)
   - nevertheless, critical out of memory conditions (where not even SSH worked) were only seen in k3s, while RKE2 systems remained accessible despite API errors
 - MariaDB is only faster than embedded etcd at small scale. MariaDB run time appears to be polynomial at large scale while etcd keeps linear. etcd is up to ~6x faster
   - Moving from a 2-CPU, 4 GB RAM instance to a 4-CPU, 16 GB RAM, provisioned IOPS MariaDB instance made little measurable difference
 - PostgreSQL is consistently faster than MariaDB although still importantly slower than etcd. Polynomal time growth is observed in PostgreSQL as well
   - Distance between PostgreSQL and etcd is less when resources are big, although etcd is still consistently faster

[Details in Excel format](https://mysuse-my.sharepoint.com/:x:/g/personal/moio_suse_com/Ee7ylp4PVz1GvDOzFpaVlR0BBpeEfjgPre2qu7_ROu0XMg?e=KF458b) are available to SUSE employees.

## Methodology notes

- no workload is running on the local cluster, no downstream clusters are registered
- every measurement is repeated 5 times and average/standard deviation is calculated. All standard deviations observed appeared to be reasonable to sustain conclusions above
- API calls under benchmark are accessed via pure REST, setup calls use a Kubernetes client
- all request responses are checked for errors (non-200 return codes). All results with errors were discarded, conclusions above use non-error cases only
- pauses are used before running each benchmark to ensure previous operations do not interfere (eg. handlers triggered asynchronously after resource creation)
- entire environment is destroyed and recreated via Terraform for each test

## Test outline
- infrastructure is set up:
    - AWS hardware (VMs, network devices, databases...) are deployed
    - either k3s or RKE2 is installed on cluster nodes, Rancher is installed on top of it
- test is conducted:
    - initial admin user is set up ([detail](../cypress/cypress/e2e/users.cy.js))
    - the benchmark script is run (see below)
    - results are collected in CSV form and elaborated in Excel

## AWS Hardware configuration outline

- bastion host (for SSH tunnelling only): `t4g.small`, 50 GiB EBS `gp3` root volume
- Rancher cluster: 3-node `t3a.xlarge` (4 vCPU, 16 GiB RAM), 50 GiB EBS `gp3` root volume
- RDS database (in MariaDB and PostgreSQL benchmarks only): `b.t4g.xlarge` (4 Gravitron vCPU, 16 GiB RAM), 20 GiB EBS data volume
- networking: one /16 AWS VPC with two /24 subnets
    - public subnet: contains the one bastion host which exposes port 22 to the Internet via security groups
    - private subnet: contains all other nodes. Traffic allowed only internally and to/from the bastion via SSH

References:
- [instance types](https://aws.amazon.com/ec2/instance-types/)
- [EBS](https://aws.amazon.com/ebs/)
- [RDS](https://aws.amazon.com/rds/)
- [VPC](https://aws.amazon.com/vpc/)

## Software configuration outline

- bastion host: SLES 15 SP4
- k3s cluster: Rancher 2.6.9 on a 3-node v1.24.6+k3s1 cluster
    - all nodes based on Ubuntu Jammy 22.04 LTS amd64
- RKE2 cluster: Rancher 2.6.9 on a 3-node v1.24.8+rke2r1 cluster
    - all nodes based on Ubuntu Jammy 22.04 LTS amd64

## Full configuration details

All infrastructure is defined via [Terraform](https://www.terraform.io/) files in the [20221128_api_load_benchmarks](https://github.com/moio/scalability-tests/tree/20221128_api_load_benchmarks/terraform) branch. Note in particular [inputs.tf](https://github.com/moio/scalability-tests/blob/20221128_api_load_benchmarks/terraform/inputs.tf) for the main parameters.
Initial configuration is driven by [Cypress](https://www.cypress.io/) files in the [cypress/e2e](https://github.com/moio/scalability-tests/tree/20221128_api_load_benchmarks/cypress/cypress/e2e) directory.
Benchmark Python scripts are available in the [util](https://github.com/moio/scalability-tests/tree/20221128_api_load_benchmarks/util) directory.

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
git clone https://github.com/moio/scalability-tests.git
cd scalability-tests
git checkout 20221128_api_load_benchmarks
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
  - edit `terraform/inputs.tf` (specifically: `ssh_private_key_path`, `ssh_public_key_path` and fields which have `# alternatives` comments)
  - edit `terraform/inputs.tf` (specifically: fields which have `# alternatives`)
  - deploy and configure infrastructure:
```shell
./util/deploy_benchmark_infrastructure.sh
```

Note that the `deploy_benchmark_infrastructure.sh` is idempotent, it will destroy and re-create the cluster if run multiple times. This makes it easier to repeat tests (possibly with different configuration).


Execute the benchmark:
```shell
./config/ssh-to-upstream-server-node-0-*.sh KUBECONFIG=/etc/rancher/k3s/k3s.yaml python3 - 0 1000 256000 10240 <./util/benchmark_k8s_secrets.py | tee -a results.csv
```

Elements of the line above have the following meaning:
 - `./config/ssh-to-upstream-server-node-0-*.sh` opens an SSH shell to the first server node of the cluster
 - `KUBECONFIG` points to the configuration file on the first cluster node. Use `KUBECONFIG=/etc/rancher/rke2/rke2.yaml` for RKE2
 - `0 1000 256000` indicate the number of resources to create before each benchmark measure - starting from 0, first step at 1000, doubling the number up to 256000
 - `10240` is the size of the secret or config map data payload in bytes
 - `./util/benchmark_k8s_secrets.py` is the benchmark script - other available options are `./util/benchmark_k8s_config_maps.py` and `./util/benchmark_steve_secrets.py`, all accept the same parameters above
 - `| tee -a results.csv` saves results into a file that can be opened in a spreadsheet editor

### Cleanup

All created infrastructure can be destroyed via:
```shell
terraform destroy -auto-approve
```
