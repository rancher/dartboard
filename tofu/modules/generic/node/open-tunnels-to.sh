#!/bin/sh
set -x

# Kill any previously created tunnels
%{ for tunnel in ssh_tunnels ~}
pkill -f 'ssh .*-o IgnoreUnknown=TofuCreatedThisTunnel.*-L ${tunnel[0]}:localhost:[0-9]+.*'
%{ endfor ~}


MAX_RETRIES=3
ATTEMPT=1
SUCCESS=false

while [ $ATTEMPT -le $MAX_RETRIES ]; do
  echo "Attempt $ATTEMPT to set up tunnels..."

# Timeout block for tunnel creation and checks
timeout 120 sh <<'EOF'
set -e
set -x

# Create tunnels
nohup ssh -o IgnoreUnknown=TofuCreatedThisTunnel \
  -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
  -i ${ssh_private_key_path} \
  -N \
  %{ for tunnel in ssh_tunnels }-L ${tunnel[0]}:localhost:${tunnel[1]} %{ endfor }\
  %{ if ssh_bastion_host != null ~}-o ProxyCommand="ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ${ssh_private_key_path} -W %h:%p ${ssh_bastion_user}@${ssh_bastion_host}"\%{ endif ~}
  ${ssh_user}@${ssh_bastion_host != null ? private_name : public_name} >/dev/null 2>&1 &

%{ for tunnel in ssh_tunnels }
echo "Waiting for tunnel ${tunnel[0]} to be up..."
while ! nc -zv localhost ${tunnel[0]}
do
  sleep 1
done
%{ endfor }
EOF

if [ $? -eq 0 ]; then
    echo "Tunnels established successfully on attempt $ATTEMPT."
    SUCCESS=true
    break
  else
    echo "Attempt $ATTEMPT failed. Retrying..."
    ATTEMPT=$((ATTEMPT + 1))
    sleep 5
  fi
done

if [ "$SUCCESS" = false ]; then
  echo "All attempts to set up tunnels failed. Exiting."
  exit 1
fi
