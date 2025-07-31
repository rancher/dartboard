#!/usr/bin/env bash

set -xe

# use data disk if available (see mount_ephemeral.sh)
if [ -d /data ]; then
  mkdir -p /data/rancher
  ln -sf /data/rancher /var/lib/rancher
fi

# https://docs.rke2.io/known_issues/#networkmanager
if systemctl status NetworkManager; then
  cat >/etc/NetworkManager/conf.d/rke2-canal.conf <<EOF
[keyfile]
unmanaged-devices=interface-name:cali*;interface-name:flannel*
EOF
  systemctl reload NetworkManager
fi

# https://docs.rke2.io/known_issues/#wicked
cat >/etc/sysctl.d/90-rke2.conf <<EOF
net.ipv4.conf.all.forwarding=1
net.ipv6.conf.all.forwarding=1
EOF

# pre-shared secrets
mkdir -p /var/lib/rancher/rke2/server/tls/
cat >/var/lib/rancher/rke2/server/tls/client-ca.key <<EOF
${client_ca_key}
EOF
cat >/var/lib/rancher/rke2/server/tls/client-ca.crt <<EOF
${client_ca_cert}
EOF
cat >/var/lib/rancher/rke2/server/tls/server-ca.key <<EOF
${server_ca_key}
EOF
cat >/var/lib/rancher/rke2/server/tls/server-ca.crt <<EOF
${server_ca_cert}
EOF
cat >/var/lib/rancher/rke2/server/tls/request-header-ca.key <<EOF
${request_header_ca_key}
EOF
cat >/var/lib/rancher/rke2/server/tls/request-header-ca.crt <<EOF
${request_header_ca_cert}
EOF

mkdir -p /etc/rancher/rke2/
cat >/etc/rancher/rke2/config.yaml <<EOF
server: ${jsonencode(server_url)}
token: ${jsonencode(token)}
%{ for label in labels ~}
node-label: ${label.key}=${label.value}
%{ endfor ~}
%{ for taint in taints ~}
node-taint: ${taint.key}=${taint.value}:${taint.effect}
%{ endfor ~}
tls-san:
%{ for san in sans ~}
  - ${jsonencode(san)}
%{ endfor ~}
kubelet-arg:
- "--config=/etc/rancher/rke2/kubelet-custom.config"
kube-controller-manager-arg: "node-cidr-mask-size=${node_cidr_mask_size}"
EOF

cat > /etc/rancher/rke2/kubelet-custom.config <<EOF
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
maxPods: ${max_pods}
EOF

cat >>/root/.bash_profile <<EOF
export PATH=\$PATH:/var/lib/rancher/rke2/bin/
export KUBECONFIG=/etc/rancher/rke2/rke2.yaml
EOF

cat >>/root/.bashrc <<EOF
export PATH=\$PATH:/var/lib/rancher/rke2/bin/
export KUBECONFIG=/etc/rancher/rke2/rke2.yaml
EOF

# installation
export INSTALL_RKE2_VERSION=${distro_version}
export INSTALL_RKE2_TYPE=${type}

curl -sfL https://get.rke2.io | sh -
systemctl enable rke2-${type}.service
systemctl restart rke2-${type}.service
