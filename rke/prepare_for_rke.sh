#!/usr/bin/env bash

set -xe

# https://rancher.com/docs/rke/latest/en/os/#red-hat-enterprise-linux-rhel-oracle-linux-ol-centos
if grep --quiet --ignore-case rhel < /etc/os-release; then
  dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
  dnf install -y docker-ce docker-ce-cli containerd.io
fi

# https://rancher.com/docs/rke/latest/en/os/#suse-linux-enterprise-server-sles-opensuse
if grep --quiet --ignore-case suse < /etc/os-release; then
  zypper addrepo https://download.docker.com/linux/centos/docker-ce.repo
  zypper install -y docker-ce
fi

systemctl enable docker
systemctl start docker
