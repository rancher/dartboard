#!/usr/bin/env bash

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

private_key_path="${1:-${script_dir}/id_rsa}"
server_address_file="${2:-${script_dir}/nodes.txt}"
downloaded_cert_key_path="${3:-${script_dir}/tls.key}"
downloaded_cert_chain_path="${4:-${script_dir}/tls.crt}"

while IFS= read -r line; do
  scp -i "${private_key_path}" "${downloaded_cert_key_path}" root@"$line":~/certs/privkey.pem
  scp -i "${private_key_path}" "${downloaded_cert_chain_path}" root@"$line":~/certs/fullchain.pem
done < "${server_address_file}"

while IFS= read -r line; do
  ssh -n -i "${private_key_path}" root@"$line" docker restart docker-nginx
done < "${server_address_file}"
