#!/bin/sh

set -euo pipefail

: ${CONTEXT:?CONTEXT must be set}
: ${KUBECONFIG:?KUBECONFIG must be set}

# get with .bin/tofu -chdir=./tofu/main/aws output -json | jq .clusters.value.upstream.kubernetes_addresses.private -r
: ${BASE_URL:?BASE_URL must be set}

RESULTS_DIR=${RESULTS_DIR:-/tmp/results}


mkdir -p "${RESULTS_DIR}"

COUNT=1000
while [ "$COUNT" -le 256000 ]
do
  echo ""
  echo "**** Creating $COUNT resources"
  echo ""

  k6 run -q -e KUBECONFIG=${KUBECONFIG} -e CONTEXT=${CONTEXT} -e BASE_URL=${BASE_URL} -e CONFIG_MAP_COUNT=$COUNT -e VUS=10 -e SECRET_COUNT=10 -e DATA_SIZE=4 /generic/create_k8s_resources.js | tee "${RESULTS_DIR}/write_${COUNT}_configmaps_results.txt"

  # cool down
  echo ""
  echo "**** Cooling down during $((COUNT/300)) seconds..."
  echo ""
  sleep $((COUNT/300))

  echo ""
  echo "**** Warming up ..."
  echo ""
  # warmup
  k6 run -q -e VUS=1 -e PER_VU_ITERATIONS=5 -e KUBECONFIG=${KUBECONFIG} -e CONTEXT=${CONTEXT} -e BASE_URL=${BASE_URL} /k8s_api_benchmark.js

  echo ""
  echo "**** Executing tests with COUNT=$COUNT ..."
  echo ""

  # test + record
  k6 run -q -e VUS=10 -e PER_VU_ITERATIONS=30 -e KUBECONFIG=${KUBECONFIG} -e CONTEXT=${CONTEXT} -e BASE_URL=${BASE_URL} /k8s_api_benchmark.js | tee "${RESULTS_DIR}/read_${COUNT}_configmaps_results.txt"
  echo ""
  echo "**** DONE $COUNT"
  echo ""

  COUNT=$((COUNT * 2))
done
