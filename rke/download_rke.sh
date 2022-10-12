#!/usr/bin/env bash

set -xe

curl --location -o ${target}/rke -z ${target}/rke https://github.com/rancher/rke/releases/download/${version}/rke_${os_platform}

chmod +x ${target}/rke
