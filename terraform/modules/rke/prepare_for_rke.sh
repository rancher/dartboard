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

# https://docs.docker.com/engine/install/ubuntu/#install-using-the-repository
if grep --quiet --ignore-case ubuntu < /etc/os-release; then
  mkdir -p /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --no-tty --yes --dearmor -o /etc/apt/keyrings/docker.gpg

  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" > /etc/apt/sources.list.d/docker.list

  export DEBIAN_FRONTEND=noninteractive
  apt-get update
  # HACK: use fixed versions RKE supports
  apt-get install --yes --allow-downgrades docker-ce=5:20.10.23~3-0~ubuntu-jammy docker-ce-cli=5:20.10.23~3-0~ubuntu-jammy containerd.io docker-compose-plugin
fi

systemctl enable docker
systemctl start docker
