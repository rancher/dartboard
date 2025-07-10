#!/bin/zsh

label=${1-"wave=1"}
CONTROL_PLANE_NODES=$(kubectl get nodes --selector='node-role.kubernetes.io/control-plane' -o jsonpath="{.items[*].status.addresses[?(@.type=='Hostname')].address}" | tr ' ' '\n' )

kubectl get pods -A -o json | jq -r '
  .items
  | group_by(.spec.nodeName)
  | map({node: .[0].spec.nodeName, count: length})
  | .[]
  | select(.count < 150) | [ .node, .count ] | @csv
' | grep -vFf <(echo $CONTROL_PLANE_NODES) | \
  sed 's/"//g' | cut -d, -f1 | \
  while read n; do kubectl label node "$n" "$label"; done
