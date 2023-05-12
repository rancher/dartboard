# Scalability tests

This repo collects code, instructions and results for scalability tests on the Rancher product family.

## Requirements
 - `docker`
 - `kubectl`
 - `helm`
 - `node`
 - [k6](https://github.com/grafana/k6/releases/tag/v0.44.1)
 - [terraform 1.3.7](https://releases.hashicorp.com/terraform/1.3.7)
 - ~10 GiB of free RAM

## Usage

```
git clone https://github.com/moio/scalability-tests.git
cd scalability-tests
git checkout 20230512_multiversion_test

cd bin
./setup.mjs
```

To teardown:
```
./teardown.mjs
```

## Troubleshooting

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
