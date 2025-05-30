#!/bin/bash

set -euo pipefail

KUBECONFIG=$(realpath ../tofu/main/aws/default_config/tester.yaml)
CONTEXT=tester
BASE_URL=https://upstream.local.gd:7445

kubectl create secret generic k6-kubeconfig --namespace tester --from-file=$(realpath ../tofu/main/aws/default_config/upstream.yaml)

helm --kubeconfig=${KUBECONFIG} upgrade --install --namespace=tester k6-files ../charts/k6-files --create-namespace

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
    image: grafana/k6:0.54.0
    name: k6shell
    resources: {}
    stdin: true
    tty: true
    volumeMounts:
      - mountPath: /k6
        name: k6-test-files
      - mountPath: /k6/lib
        name: k6-lib-files
      - mountPath: /kubeconfig
        name: k6-kubeconfig
  volumes:
    - name: k6-test-files
      configMap:
        name: k6-test-files
    - name: k6-lib-files
      configMap:
        name: k6-lib-files
    - name: k6-kubeconfig
      secret:
        secretName: k6-kubeconfig
EOF

kubectl attach k6shell -c k6shell -n tester -i -t


# kubectl cp -n tester k6shell:/home/k6/results ../docs/20250513\ -\ Kubernetes\ API\ benchmark\ test\ results
