#!/usr/bin/env bash

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

private_key_path="${1:-${script_dir}/id_rsa}"
server_address_file="${2:-${script_dir}/nodes.txt}"

while IFS= read -r line; do
  ssh-copy-id -i "${private_key_path}" root@"$line"
done < "${server_address_file}"
