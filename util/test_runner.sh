#!/bin/bash

export KUBECONFIG=/kubeconfig/upstream.yaml
export CONTEXT=upstream
# get with .bin/tofu -chdir=./tofu/main/aws output -json | jq .clusters.value.upstream.kubernetes_addresses.private -r
export BASE_URL=https://ip-172-16-1-63.ec2.internal:6443


cd ~

mkdir results

COUNT=1000
while [ "$COUNT" -le 256000 ]
do
  k6 run -e KUBECONFIG=${KUBECONFIG} -e CONTEXT=${CONTEXT} -e BASE_URL=${BASE_URL} -e CONFIG_MAP_COUNT=$COUNT -e VUS=10 -e SECRET_COUNT=10 -e DATA_SIZE=4 /k6/create_k8s_resources.js | tee results/write_${COUNT}_configmaps_results.txt

  # cool down
  sleep $((COUNT/300))

  # warmup
  k6 run -e VUS=1 -e PER_VU_ITERATIONS=5 -e KUBECONFIG=${KUBECONFIG} -e CONTEXT=${CONTEXT} -e BASE_URL=${BASE_URL} /k6/k8s_api_benchmark.js

  # test + record
  k6 run -e VUS=10 -e PER_VU_ITERATIONS=30 -e KUBECONFIG=${KUBECONFIG} -e CONTEXT=${CONTEXT} -e BASE_URL=${BASE_URL} /k6/k8s_api_benchmark.js | tee results/read_${COUNT}_configmaps_results.txt
  echo "**** DONE $COUNT"

  COUNT=$((COUNT * 2))
done
