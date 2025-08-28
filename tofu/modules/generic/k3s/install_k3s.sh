#!/usr/bin/env bash

set -xe

# use data disk if available (see mount_ephemeral.sh)
if [ -d /data ]; then
  mkdir -p /data/rancher
  ln -sf /data/rancher /var/lib/rancher
fi

# pre-shared secrets
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
%{ if server_url != null }
server: ${jsonencode(server_url)}
%{ endif ~}
%{ if token != null }
token: ${jsonencode(token)}
%{ endif ~}
%{ if cluster_init ~}
cluster-init: true
%{ endif ~}
%{ for label in labels ~}
node-label: ${label.key}=${label.value}
%{ endfor ~}
%{ for taint in taints ~}
node-taint: ${taint.key}=${taint.value}:${taint.effect}
%{ endfor ~}
%{ if exec == "server" ~}
tls-san:
%{ for san in sans ~}
  - ${jsonencode(san)}
%{ endfor ~}
kube-controller-manager-arg: "node-cidr-mask-size=${node_cidr_mask_size}"
%{ endif ~}
kubelet-arg: "config=/etc/rancher/k3s/kubelet-custom.config"
%{ if datastore_endpoint != null ~}
datastore-endpoint: "${datastore_endpoint}"
%{ endif ~}
EOF

cat > /etc/rancher/k3s/kubelet-custom.config <<EOF
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
maxPods: ${max_pods}
EOF

# installation
export INSTALL_K3S_VERSION=${distro_version}
export INSTALL_K3S_EXEC=${exec}

MAX_RETRIES=5
RETRY_DELAY=5 # seconds
# Default to a failure status
status=1
for (( i=1; i<=MAX_RETRIES; i++ )); do
  if [ -f "${get_k3s_path}" ]; then
      sh /tmp/get_k3s.sh
      status=$?
  else
      curl -sfL https://get.k3s.io | sh -
      status=$?
  fi

  if [ $status -eq 0 ]; then
        break # Exit the loop if the script run was successful
  else
      echo "Installation failed. Retrying in $RETRY_DELAY seconds..."
      sleep "$RETRY_DELAY"
  fi
done

if [ $i -gt $MAX_RETRIES ]; then
    echo "Command failed after $MAX_RETRIES attempts."
    exit 1
fi
