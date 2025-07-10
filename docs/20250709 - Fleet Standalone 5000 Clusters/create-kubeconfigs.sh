#!/bin/zsh

ns=${1-k3k-downstream001}

while IFS=, read -r name port san
do
    k3kcli kubeconfig generate --namespace "$ns" --name "$name" --kubeconfig-server "$san"
    sed -i -e "s/ec2.internal:.*/ec2.internal:$port/; s/default/$name/" "$ns-$name"-kubeconfig.yaml
done < <(kubectl get clusters -n "$ns" -ojsonpath='{range .items[*]}{.metadata.name}{","}{.spec.expose.nodePort.serverPort}{","}{.spec.tlsSANs[1]}{"\n"}{end}')
