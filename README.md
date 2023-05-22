# Scalability tests

This repo collects code, instructions and results for scalability tests on the Rancher product family.

## Usage

```
cd terraform
terraform init
terraform apply -auto-approve
```

npx cypress open --config watchForFileChanges=false

## Troubleshooting

### k3d: cluster not created

If you get this error from `terraform apply`:
```
Error: Failed Cluster Start: Failed to add one or more agents: Node k3d-... failed to get ready: error waiting for log line `successfully registered node` from node 'k3d-moio-upstream-agent-0': stopped returning log lines: node k3d-... is running=true in status=restarting
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

If you get this error from `terraform apply`:
```
│ Error: Kubernetes cluster unreachable: Get "https://upstream.local.gd:6443/version": dial tcp 127.0.0.1:6443: connect: connection refused
```

SSH tunnels might be broken. Reopen them via:
```shell
./config/open-tunnels-to-upstream-*.sh
```

### Terraform extended logging

In case Terraform returns an error with little context about what happened, use the following to get more complete debugging output:
```shell
export TF_LOG=debug
```

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
