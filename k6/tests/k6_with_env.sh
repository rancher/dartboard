#!/usr/bin/env bash

env_file=${1:-"./.env"} # path to the source-able .env file containing relevant variables
test_file=${2}          # path to the k6 test file to run
getRancherLogs=${3:-"false"}   # whether or not to get rancher pod logs
iters=${4:-1}                  # the number of iterations of the test to loop through
delay=${5:-15}                 # the delay between iterations, in minutes
address=${6:-"localhost:6565"} # the domain:PORT address to the API server https://grafana.com/docs/k6/latest/using-k6/k6-options/reference/#address

counter=1
sleepDuration=$((delay * 60))
while [ ${counter} -le "${iters}" ]; do
  iterStart=$(date -u '+%FT%T%:z')
  echo "Started iteration at: ${iterStart}"
  # shellcheck disable=SC1090
  source "${env_file}" && k6 run --summary-mode=full -a "${address}" --out json="${test_file%.js*}-output${counter}.json" "${test_file}"
  if [ "${getRancherLogs}" = "true" ]; then
    echo "${KUBECONFIG}"
    kubectl -n cattle-system logs -l status.phase=Running -l app=rancher -c rancher --timestamps --since-time="${iterStart}" --tail=999999 >"rancher_logs-test_${test_file%.js*}.txt"
  fi
  if [[ ${iters} -ge 2 ]]; then
    sleep ${sleepDuration}
  fi
  counter=$((counter + 1))
done
