#!/bin/zsh

prefix=$1
wait_at=100
count=0

for cfg in k3k*yaml; do
    count=$(( count + 1 ))

    cluster=${cfg#k3k-downstream}
    cluster=${cfg%-kubeconfig.yaml}
    value=$(cat "$cfg")

    clustername=$prefix-$cluster
    kubectl create secret generic -n fleet-default kcfg-$clustername --from-literal=value="$value"

    kubectl apply -n fleet-default -f - <<EOF
apiVersion: "fleet.cattle.io/v1alpha1"
kind: Cluster
metadata:
  name: $clustername
  namespace: fleet-default
  labels:
    name: $cluster
    cluster: $prefix
spec:
  kubeConfigSecret: kcfg-$clustername
EOF


    if (( count % wait_at == 0 )); then
        echo "sleeping after $count agents"
        sleep 100
    fi

done
