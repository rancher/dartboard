#!/usr/bin/env bash

set -xe

arch=`uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/'`
mkdir -p /tmp/rke2-artifacts
pushd /tmp/rke2-artifacts
  version=$(echo "${distro_version}" | sed 's/+/%2B/g')
  wget -c https://prime.ribs.rancher.io/rke2/"$version"/rke2-images-core.linux-"$arch".tar.gz
  wget -c https://prime.ribs.rancher.io/rke2/"$version"/rke2-images-canal.linux-"$arch".tar.gz
  wget -c https://prime.ribs.rancher.io/rke2/"$version"/rke2-images-calico.linux-"$arch".tar.gz
  wget -c https://prime.ribs.rancher.io/rke2/"$version"/rke2.linux-"$arch".tar.gz
  wget -c https://prime.ribs.rancher.io/rke2/"$version"/sha256sum-"$arch".txt
popd

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
%{ if server_url != null }
server: ${jsonencode(server_url)}
%{ endif ~}
%{ if token != null }
token: ${jsonencode(token)}
%{ endif ~}
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
system-default-registry: registry.rancher.com
EOF

cat > /etc/rancher/rke2/kubelet-custom.config <<EOF
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
maxPods: ${max_pods}
EOF

cat > /etc/sysctl.d/99-neigh.conf <<EOF
net.ipv4.neigh.default.gc_thresh3 = 4096
net.ipv4.neigh.default.gc_thresh2 = 2048
net.ipv4.neigh.default.gc_thresh1 = 1024
EOF
sysctl -p /etc/sysctl.d/99-neigh.conf

cat > /etc/sysctl.d/99-inotify.conf <<EOF
fs.inotify.max_user_instances = 8192
fs.inotify.max_user_watches = 524288
EOF
sysctl -p /etc/sysctl.d/99-inotify.conf

# can be used for longhorn or rancher/local-path-provisioner
mkdir -p /data/storage

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

MAX_RETRIES=5
RETRY_DELAY=5 # seconds
# Default to a failure status
status=1
for (( i=1; i<=MAX_RETRIES; i++ )); do
  if [ -f "${get_rke2_path}" ]; then
      cat ${get_rke2_path} > install.sh
      status=$?
  else
      curl -sfL https://get.rke2.io --output install.sh
      status=$?
  fi

  sudo -s INSTALL_RKE2_ARTIFACT_PATH=/tmp/rke2-artifacts sh install.sh
  status=$?

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

systemctl enable rke2-${type}.service
systemctl restart rke2-${type}.service
