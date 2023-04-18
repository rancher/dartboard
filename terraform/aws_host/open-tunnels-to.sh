#!/bin/sh

# Kill any previously created tunnels
%{ for tunnel in ssh_tunnels }
pkill -f 'ssh .*-o IgnoreUnknown=TerraformCreatedThisTunnel.*-L ${tunnel[0]}:localhost:[0-9]+.*'
%{ endfor }

# Create tunnels
nohup ssh -o IgnoreUnknown=TerraformCreatedThisTunnel \
  -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
  -N \
  %{ for tunnel in ssh_tunnels }-L ${tunnel[0]}:localhost:${tunnel[1]} %{ endfor }\
  %{ if ssh_bastion_host != null ~}-o ProxyCommand="ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -W %h:%p root@${ssh_bastion_host}"\%{ endif ~}
  root@${ssh_bastion_host != null ? private_name : public_name} >/dev/null 2>&1 &

%{ for tunnel in ssh_tunnels }
echo "Waiting for tunnel ${tunnel[0]} to be up..."
while ! nc -zv localhost ${tunnel[0]}
do
  sleep 1
done
%{ endfor }
