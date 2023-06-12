#!/usr/bin/env bash

set -xe

curl --location -o ${target}/rke -z ${target}/rke https://github.com/rancher/rke/releases/download/${version}

chmod +x ${target}/rke
