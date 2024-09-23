# Dartboard

A tool to run scalability and performance tests on the Rancher product family.

Supports deploying to AWS, Azure, OVH OpenStack and bare metal servers (via SSH) for infrastructure; k3s, RKE2, k3d and AKS as Kubernetes distributions; any recent version of Rancher.

## Usage

 - create a Definition of Alacritous and Repeatable Test (or **dart**) YAML file by adapting one of the examples in [darts](./darts)
 - `dartboard deploy --dart=./darts/my_dart.yaml` will:
   - deploy (virtual) infrastructure via [OpenTofu](https://opentofu.org/): the cluster Rancher runs on ("upstream"), clusters managed by Rancher ("downstream") and a special "tester" cluster where load generation and monitoring software runs
   - deploy and configure Rancher
   - execute load tests via [k6](https://k6.io/)
 - `dartboard destroy` destroys all infrastructure

Special cases:
 - `dartboard apply` only runs `tofu apply` without configuring any software (Rancher, load generation, moniroting...)
 - `dartboard load` only runs k6 load tests assuming Rancher has already been deployed
 - `dartboard get-access` returns details to access the created clusters and applications

To recreate environments:
 - `dartboard reapply` runs `destroy` and then `apply`, tearing down and recreating test configuration infrastructure without any software (Rancher, load generation, moniroting...)
 - `dartboard redeploy` runs `destroy` and then `deploy`, tearing down and recreating the full environment, infrastructure and software (use this if unsure)

## Installation

Requirements:
 - [OpenTofu](https://opentofu.org/)
 - kubectl
 - [k6](https://k6.io/)
 - [k3d](https://k3d.io) (only needed for k3d)
 - [Docker](https://www.docker.com/) (only for k3d)

Clone the repository and run `make`.

## Test history

See the [docs](docs) directory for a list of tests that were run with previous versions of this code and their results.

## Common Troubleshooting

### k3d: cluster not created

If you get this error from `tofu apply`:
```
Error: Failed Cluster Start: Failed to add one or more agents: Node k3d-... failed to get ready: error waiting for log line `successfully registered node` from node 'k3d-st-upstream-agent-0': stopped returning log lines: node k3d-... is running=true in status=restarting
```

And `docker logs` on the node container end with:
```
kubelet.go:1361] "Failed to start cAdvisor" err="inotify_init: too many open files"
```

Then you might need to increase inotify's maximum open file counts via:
```
echo 256 > /proc/sys/fs/inotify/max_user_instances
echo "fs.inotify.max_user_instances = 256" > /etc/sysctl.d/99-inotify-mui.conf
```

### Kubernetes cluster unreachable

If you get this error from `tofu apply`:
```
│ Error: Kubernetes cluster unreachable: Get "https://upstream.local.gd:6443/version": dial tcp 127.0.0.1:6443: connect: connection refused
```

SSH tunnels might be broken. Reopen them via:
```shell
./config/open-tunnels-to-upstream-*.sh
```

### OpenTofu extended logging

In case OpenTofu returns an error with little context about what happened, use the following to get more complete debugging output:
```shell
export TF_LOG=debug
```

### Forcibly stopping all SSH tunnels

```shell
pkill -f 'ssh .*-o IgnoreUnknown=TofuCreatedThisTunnel.*'
```

### Troubleshooting inaccessible Azure VMs

If an Azure VM is not accessible via SSH, try the following:
- add the `boot_diagnostics = true` option in `inputs.tf`
- apply or re-deploy
- in the Azure Portal, click on Home -> Virtual Machines -> <name> -> Help -> Reset Password 
- then Home -> Virtual Machines -> <name> -> Help -> Serial Console

That should give you access to the VM's console, where you can log in with the new password and troubleshoot.

## Tips

### Use k3d targeting a remote machine running the Docker daemon

Use the following command to point to a remote Docker host:
```shell
export DOCKER_HOST=tcp://remotehost:remoteport
```

Note that the host has to have TCP socket enabled, in addition or replacing the default Unix socket.

Eg. on SUSE OSs edit the `/etc/sysconfig/docker` file as root and add or edit the following line:
```
DOCKER_OPTS="-H unix:///var/run/docker.sock -H tcp://127.0.0.1:2375"
```

If you access the Docker host via SSH, you might want to forward the Docker port along with any relevant ports to access Rancher and the clusters' Kubernetes APIs, for example:

```shell
ssh remotehost -L 2375:localhost:2375 -L 8443:localhost:8443 $(for KUBEPORT in $(seq 6445 1 6465); do echo " -L ${KUBEPORT}:localhost:${KUBEPORT}" ; done | tr -d "\n")
```

### Use custom-built Rancher images on k3d

When using `k3d`, change `RANCHER_IMAGE_TAG` and if an image with the same tag is found it will be added to relevant clusters.

This is useful during Rancher development to test Rancher changes on k3d clusters.

## Passing custom OpenTofu variables

OpenTofu variables can be overridden using `TOFU_VAR_FILE` environment variable, to point to a [`.tfvars` file](https://developer.hashicorp.com/tofu/language/values/variables#variable-definitions-tfvars-files). The variable should contain a path to the file in json or tfvars format.
For example, for the `ssh` module, nodes' ip addresses, login name, etc. can be overridden as follows:

```shell
export TOFU_WORK_DIR=tofu/main/ssh
export TOFU_VAR_FILE=tofu/examples/ssh.tfvars.json
```
