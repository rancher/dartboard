# Dartboard

A tool to run scalability and performance tests on the Rancher product family.

Supports deploying to AWS and Azure or a local Docker daemon (via k3d) for infrastructure; k3s, RKE2 and AKS as Kubernetes distributions; any recent version of Rancher.

## Usage

 - create a Definition of Alacritous and Repeatable Test (or **dart**) YAML file by adapting one of the examples in [darts](./darts)
 - `dartboard deploy --dart=./darts/my_dart.yaml` will:
   - deploy (virtual) infrastructure via [OpenTofu](https://opentofu.org/): the cluster Rancher runs on ("upstream"), clusters managed by Rancher ("downstream") and a special "tester" cluster where load generation and monitoring software runs
   - deploy and configure Rancher
   - execute load tests via [k6](https://k6.io/)
 - `dartboard destroy` destroys all infrastructure

Special cases:
 - `dartboard apply` only runs `tofu apply` without configuring any software (Rancher, load generation, monitoring...)
 - `dartboard load` only runs k6 load tests assuming Rancher has already been deployed
 - `dartboard get-access` returns details to access the created clusters and applications

To recreate environments:
 - `dartboard reapply` runs `destroy` and then `apply`, tearing down and recreating test configuration infrastructure without any software (Rancher, load generation, moniroting...)
 - `dartboard redeploy` runs `destroy` and then `deploy`, tearing down and recreating the full environment, infrastructure and software (use this if unsure)

### "Bring Your Own" AWS VPC
There is some manual configuration required in order to use an existing AWS VPC instead of having the tofu modules create a full set of networking resources.

1. Have an existing VPC with a DHCP options set configured so that DNS = "AmazonProvidedDNS".
2. Create three subnets, requirements are as follows:
   1. One subnet should contain the substring "public" (case-sensitive), and should be tagged with `Tier = Public` (case-sensitive)
   2. One subnet should contain the substring "private" (case-sensitive), and should be tagged with `Tier = Private` (case-sensitive)
   3. One subnet should contain the substring "secondary-private" (case-sensitive), and should be tagged with `Tier = SecondaryPrivate` (case-sensitive)
   4. Each subnet should be assigned to the VPC you intend to use

Once these resources are manually setup, you can set the `existing_vpc_name` tofu variable in your Dart file and deploy as you normally would.

## Installation

Download and unpack a [release](https://github.com/rancher/dartboard/releases/), it's a self-contained binary.

### Usage with Docker

Alternatively, you can use the provided `Dockerfile` to build a container image that includes `dartboard`, `k6`, and all necessary dependencies. This is a great way to ensure a consistent environment while testing changes.

#### 1. Build the Docker Image

From the root of the repository, run:

```bash
docker build -t dartboard:latest .
```

#### 2. Running `dartboard` Commands

The container's entrypoint is set to `dartboard`, so you can run subcommands directly. You'll need to mount your project directory and pass any necessary credentials as environment variables.

**Example: Deploy an environment**
```bash
docker run --rm -it \
  -v "$(pwd):/dartboard" \
  --workdir /dartboard \
  -e AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY \
  dartboard:latest --dart ./darts/aws.yaml deploy
```

*   `--rm`: Removes the container after it exits.
*   `-v "$(pwd):/dartboard"`: Mounts the current directory (in this case, the project root) into `/dartboard` inside the container, making your DART files and configurations accessible.
*   `--workdir /dartboard`: Sets the working directory inside the container.
*   `-e AWS_*`: Passes your local AWS credentials into the container.
*   `--dart ./darts/aws.yaml`: Must go before the subcommand being used, and defines the DART file to be used when running the subcommand.
*   NOTE: You may need to mount additional files/directories depending on the module in use and which DART file options are being used.

#### 3. Running `k6` Tests

Since the image is based on `grafana/k6`, you can also use it to run `k6` tests by overriding the entrypoint. This is useful for running performance tests against your deployed infrastructure.

**Example: Run a k6 test script**
```bash
cd k6
docker run --rm -it \
  --entrypoint k6 \
  -v "$(pwd)/k6:/home/k6" \
  -e BASE_URL="https://your-rancher-url" \
  -e TOKEN="your-api-token" \
  dartboard:latest run /home/k6/tests/api_benchmark.js
```

*   `--entrypoint k6`: Overrides the default `dartboard` entrypoint to run the `k6` binary instead.
*   `-v "$(pwd)/k6:/home/k6"`: Mounts the current directory (in this case, local `k6` scripts directory)  into the container.
*   `-e ...`: Passes environment variables required by your k6 script, such as the target URL and authentication tokens.

## Jenkins Jobs

*   `dartboard-deploy`: This is the main job for deploying a full test environment.
*   `dartboard-k6-runner`: This job is used to execute a k6 performance test script against an existing environment using the `dartboard` container.

### Build Summary

After a `dartboard-deploy` job completes, it archives a `summary.html` file. This summary is the best place to find information about the
deployed environment and useful artifacts for managing and using the environment.

You can find this summary in the form of a `.html` file on the build's page in Jenkins.

## Test history

See the [docs](docs) directory for a list of tests that were run with previous versions of this code and their results.

## Qase k6 Reporter

The `qasereporter-k6` utility is a command-line tool included as a separate golang module that parses the output of a k6 test run and reports the results to a test case wtihin a Qase test run. It can be used in CI/CD pipelines to automatically update test cases in Qase with the results from k6 performance tests.

For detailed usage instructions, including environment variables and command-line flags, please see the [qasereporter-k6 README](qasereporter-k6/README.md).

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

## Harvester: bypassing TLS verification

If you get the following error:

```
Error: Get "https://$ADDRESS/k8s/clusters/local/apis/harvesterhci.io/v1beta1/settings/server-version": tls: failed to verify certificate
```

Then your Harvester installation's TLS certificate is not set up correctly, or trusted by your system. Ideally, address those issues, otherwise communication with Harvester will not be secure.

If you want to bypass TLS checks edit your kubeconfig file to remove the `certificate-authority-data` entry and add a `insecure-skip-tls-verify: true` entry instead.
