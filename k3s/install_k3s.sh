#!/usr/bin/env bash

set -xe

export INSTALL_K3S_VERSION=${k3s_version}


mkdir -p /var/lib/rancher/k3s/server/tls/
cat >/var/lib/rancher/k3s/server/tls/client-ca.key <<EOF
${client_ca_key}
EOF
cat >/var/lib/rancher/k3s/server/tls/client-ca.crt <<EOF
${client_ca_cert}
EOF
cat >/var/lib/rancher/k3s/server/tls/server-ca.key <<EOF
${server_ca_key}
EOF
cat >/var/lib/rancher/k3s/server/tls/server-ca.crt <<EOF
${server_ca_cert}
EOF
cat >/var/lib/rancher/k3s/server/tls/request-header-ca.key <<EOF
${request_header_ca_key}
EOF
cat >/var/lib/rancher/k3s/server/tls/request-header-ca.crt <<EOF
${request_header_ca_cert}
EOF

mkdir -p /etc/rancher/k3s/
cat >/etc/rancher/k3s/config.yaml <<EOF
tls-san:
  - ${yamlencode(name)}
EOF

curl -sfL https://get.k3s.io | sh -
