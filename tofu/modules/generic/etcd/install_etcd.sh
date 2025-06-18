#!/usr/bin/env bash

set -xe

ETCD_VER=${etcd_version}
DOWNLOAD_URL=https://github.com/etcd-io/etcd/releases/download

curl -L $DOWNLOAD_URL/$ETCD_VER/etcd-$ETCD_VER-linux-amd64.tar.gz -o /tmp/etcd-$ETCD_VER-linux-amd64.tar.gz
tar xzvf /tmp/etcd-$ETCD_VER-linux-amd64.tar.gz -C /usr/bin --strip-components=1

# use data disk if available (see mount_ephemeral.sh)
if [ -d /data ]; then
  mkdir -p /data/etcd
  ln -sf /data/etcd /var/lib/etcd
fi

cat >/etc/systemd/system/etcd.service <<EOF
[Unit]
Description=etcd
StartLimitIntervalSec=60
StartLimitBurst=10

[Service]
ExecStart=/usr/bin/etcd \
  --name ${etcd_name} \
  --listen-peer-urls http://${server_ip}:2380 \
  --listen-client-urls http://${server_ip}:2379,http://127.0.0.1:2379 \
  --advertise-client-urls http://${server_ip}:2379 \
  --initial-advertise-peer-urls http://${server_ip}:2380 \
  --initial-cluster-token ${etcd_token} \
  --initial-cluster ${join(",", formatlist("%s=http://%s:2380", etcd_names, server_names))} \
  --initial-cluster-state new \
  --data-dir /var/lib/etcd

Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload

# Start kine
systemctl enable etcd
systemctl start etcd
