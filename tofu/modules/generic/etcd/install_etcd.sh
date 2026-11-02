#!/usr/bin/env bash

set -xe

DOWNLOAD_SHASUM="${etcd_shasum}"
DOWNLOAD_URL="https://github.com/etcd-io/etcd/releases/download/${etcd_version}/etcd-${etcd_version}-linux-amd64.tar.gz"

verify_sha256_digest() {
	local archive="$1"
	local checksum_file="$2"
	local expected actual

	expected=$(tr -d '\r\n' < "$${checksum_file}")

	if command -v sha256sum >/dev/null 2>&1; then
		actual=$(sha256sum "$${archive}" | awk '{print $1}')
	elif command -v shasum >/dev/null 2>&1; then
		actual=$(shasum -a 256 "$${archive}" | awk '{print $1}')
	else
		echo "No SHA256 tool found (expected sha256sum or shasum)" >&2
		exit 1
	fi

	if [[ "$${actual}" == "$${expected}" ]]; then
		echo "SHA256 verification succeeded for $${archive}"
	else
		echo "SHA256 verification FAILED for $${archive}" >&2
		exit 1
	fi
}

rm -f "/tmp/etcd.tar.gz"
rm -rf /tmp/etcd-download-test && mkdir -p /tmp/etcd-download-test

curl -L "$DOWNLOAD_URL" -o /tmp/etcd.tar.gz
verify_sha256_digest "/tmp/etcd.tar.gz" <(echo "$DOWNLOAD_SHASUM  /tmp/etcd.tar.gz")
tar xzvf /tmp/etcd.tar.gz -C /usr/bin --strip-components=1 --no-same-owner

rm -f /tmp/etcd.tar.gz

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
  --initial-cluster ${join(",", formatlist("%s=http://%s:2380", etcd_names, server_ips))} \
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
