# Developer notes

## Overall architecture

 - Terraform is used to deploy infrastructure. That includes all is necessary in order to launch Kubernetes clusters - modules should conclude producing a kubeconfig file and context
   - `tf` files in `terraform/main/` specify whole testing environments
   - `tf` files in `terraform/modules/` implement components (platform-specific or platform-agnostic)
 - the `bin/setup.mjs ` node.js script runs Terraform to create Kubernetes clusters, then Helm/kubectl to deploy and configure software under test (Rancher and/or any other component). It is designed to be idempotent
 - the `bin/run_tests.mjs ` node.js script runs `k6` scripts in `k6/`, generating load. It is designed to be idempotent
 - a Mimir-backed Grafana instance in an own cluster displays results and acts as long-term result storage

## Porting Terraform files to new platforms

 - create a new `terraform/main` subdirectory copying over `tf` files from `aws`
 - edit `inputs.tf` to include any platform-specific information
 - edit `main.tf` to use platform-specific providers, add modules as appropriate
   - platform-specific modules are prefixed with the platform name (eg. `terraform/modules/aws_*`)
   - platform-agnostic modules are not prefixed
   - platform-specific wrappers are normally created for platform-agnostic modules (eg. `aws_k3s` wraps `k3s`)
 - adapt `outputs.tf` - please note the exact structure is expected by scripts in `bin/` - change with care

It is assumed all created clusters will be able to reach one another with the same domain names, from the same network. That network might not be the same network of the machine running Terraform.

Created clusters may or may not be directly reachable from the machine running Terraform. In the current `aws` implementation, for example, all access goes through an SSH bastion host and tunnels, but that is an implementation detail and may change in future. For new platforms there is no requirement - clusters might be directly reachable with an Internet-accessible FQDN, or be behind a bastion host, Tailscale, Boundary or other mechanism. Structures in `outputs.tf` have been designed to accommodate for all cases, in particular:
 - `local_` variables refer to domain names and ports as used by the machine running Terraform,
 - `private_` variables refer to domain names and ports as used by the clusters in their network,
 - values may coincide.

`node_access_commands` are an optional convenience mechanism to allow a user to SSH into a particular node directly.

A particular deployment platform can be selected using `TERRAFORM_WORK_DIR` environment variable, eg.

```shell
export TERRAFORM_WORK_DIR=terraform/main/aws
./bin/teardown.mjs && ./bin/setup.mjs && ./bin/run_tests.mjs
```

See `terraform/main` subdirectories for the currently available platforms.

## Passing custom Terraform variables

Terraform variables can be overridden using `TERRAFORM_VAR_FILE` environment variable, to point to a [`.tfvars` file](https://developer.hashicorp.com/terraform/language/values/variables#variable-definitions-tfvars-files). The variable should contain a full path to the file in json or tfvars format.
For example, for the `ssh` module, nodes' ip addresses, login name, etc. can be overridden cas follows:

```shell
export TERRAFORM_WORK_DIR=terraform/main/ssh
export TERRAFORM_VAR_FILE=$PWD/terraform/examples/ssh.tfvars.json
./bin/setup.mjs
./bin/run_tests.mjs
./bin/teardown.mjs
```

Example files can be found in [terraform/examples].
