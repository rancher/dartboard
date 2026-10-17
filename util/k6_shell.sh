#!/bin/bash

set -euo pipefail

cd $(dirname $0)/..

TOFU_MAIN_DIRECTORY=./tofu/main/aws
if [ -n "${DART:-}" ]; then
  TOFU_MAIN_DIRECTORY="$(yq .tofu_main_directory -r $DART)"
fi

export KUBECONFIG=$(realpath ${TOFU_MAIN_DIRECTORY}/default_config/tester.yaml)
BASE_URL=$(.bin/tofu -chdir=${TOFU_MAIN_DIRECTORY} output -json | jq .clusters.value.upstream.kubernetes_addresses.private -r)

kubectl create ns tester || true
kubectl create secret generic k6shell-config --namespace tester \
  --from-file=kubeconfig=$(realpath ${TOFU_MAIN_DIRECTORY}/default_config/upstream.yaml) \
  --from-file=test_runner.sh=$(realpath util/test_runner.sh) \
  --dry-run=client -o yaml | kubectl apply -f -

helm --kubeconfig=${KUBECONFIG} upgrade --install --namespace=tester k6-files charts/k6-files --create-namespace

k6_mounts=""
for f in $(find k6 -type f | cut -d/ -f2-); do
  k6_mounts="${k6_mounts}
    - mountPath: $f
      name: k6-test-files
      subPath: ${f//\//__}"
done

kubectl -n tester delete po k6shell --wait
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: k6shell
  name: k6shell
  namespace: tester
spec:
  containers:
  - command:
    - /bin/sh
    args:
    - "-c"
    - "sh /config/test_runner.sh; echo 'You can now run: kubectl cp -n tester k6shell:/results ./results'; sleep infinity"
    image: grafana/k6:0.54.0
    name: k6shell
    resources: {}
    stdin: true
    tty: true
    env:
    - name: K6_NO_COLOR
      value: true
    - name: CONTEXT
      value: upstream
    - name: KUBECONFIG
      value: /config/kubeconfig
    - name: BASE_URL
      value: "${BASE_URL}"
    - name: RESULTS_DIR
      value: /results
    volumeMounts:${k6_mounts}
    - mountPath: /config
      name: config
    - mountPath: /results
      name: results
  volumes:
  - name: k6-test-files
    configMap:
      name: k6-test-files
  - name: kube
    secret:
      secretName: k6shell-config
  - name: results
    emptyDir: {}
EOF

kubectl wait --for=condition=Ready --for=create -n tester pod/k6shell

# kubectl exec k6shell -c k6shell -n tester -i -t -- sh
kubectl logs k6shell -c k6shell -n tester -f
