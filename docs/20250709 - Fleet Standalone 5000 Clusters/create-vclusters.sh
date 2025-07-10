#!/bin/zsh

set -e

start=${1-1}
end=${2-10}
wave=${3-1}


CONTROL_PLANE_NODES=$(kubectl get nodes --selector='node-role.kubernetes.io/control-plane' -o jsonpath="{.items[*].status.addresses[?(@.type=='Hostname')].address}" | tr ' ' '\n' )
if [ -z "$CONTROL_PLANE_NODES" ]; then
    echo "No control-plane nodes found."
    exit 1
fi

# NOTE adapt this selector to node labels, e.g.: ...control-plane,cpu=16
NODES=$(kubectl get nodes --selector='!node-role.kubernetes.io/control-plane,wave='"$wave" -o jsonpath="{.items[*].status.addresses[?(@.type=='Hostname')].address}" | tr ' ' '\n')
nodes=(${(@f)NODES})
nodeN=${#nodes[@]}

if [ "$nodeN" = 0 ]; then
    echo "No labeled (wave=$wave) agent nodes found."
    exit 1
fi

echo "Found $nodeN nodes with label wave=$wave"
sleep 5

wait_after=$nodeN
current=0

func create() {
    local i=$1

    n=$(printf "%04d" i)
    name="downstream$n"

    # network
    cpnode=$( echo $CONTROL_PLANE_NODES | shuf -n 1)
    fqdn="$cpnode.ec2.internal"

    port=$(( 30000  + i ))

    j=$(( i / 100 + 1 ))
    ns=$(printf "k3k-downstream%03d" j)
    kubectl create ns "$ns" &> /dev/null || true

    # placement
    local j=$(( i % nodeN + 1 ))
    local node=${nodes[j]}

    kubectl apply -n "$ns" -f- <<EOF
apiVersion: k3k.io/v1alpha1
kind: Cluster
metadata:
  name: $name
spec:
  expose:
    nodePort: 
      serverPort: $port
      etcdPort: 0
  #mode: virtual
  #servers: 1
  nodeSelector:
    kubernetes.io/hostname: $node
  persistence:
    type: dynamic
    storageRequestSize: 10Gi
  serverArgs:
    - --tls-san=$n.local.gd
    - --tls-san=$fqdn
  tlsSANs:
    - $n.local.gd
    - $fqdn
EOF

    echo "$name,$fqdn,$port" >> clusters.csv


    if (( i % wait_after == 0 )); then
        current=$((current + 1))
        while ! kubectl get pod -n "$ns" k3k-$name-server-0 &> /dev/null; do
            echo "waiting for pod for $name"
            sleep 5
        done
        echo "waiting for pod to be ready"
        kubectl wait --timeout=15m -n "$ns" --for=condition=Ready "pod/k3k-$name-server-0"
    fi

}

for i in $(seq "$start" "$end"); do
    create "$i"
done
