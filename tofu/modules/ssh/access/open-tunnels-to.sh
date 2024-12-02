#!/bin/sh

# Kill any previously created tunnels
%{ for tunnel in ssh_tunnels ~}
pkill -f 'ssh .*-o IgnoreUnknown=TofuCreatedThisTunnel.*-L ${tunnel[0]}:localhost:[0-9]+.*'
%{ endfor ~}

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
