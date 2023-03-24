#!/bin/bash

DOWNSTREAM_CLUSTER=`kubectl --context=k3d-moio-upstream get clusters.management.cattle.io --no-headers -o custom-columns=":metadata.name" | grep -v local`

for TAG in baseline vai
do
  kubectl --context=k3d-moio-upstream set image -n cattle-system deployment/rancher rancher=rancher/rancher:$TAG
  kubectl --context=k3d-moio-downstream set image -n cattle-system deployment/cattle-cluster-agent cluster-register=rancher/rancher-agent:$TAG
  sleep 300

  for COUNT in 100 400 1200
  do
    for CONTEXT in k3d-moio-upstream k3d-moio-downstream
    do
      k6 run -e KUBECONFIG=`realpath ~/.kube/config` -e CONTEXT=$CONTEXT -e COUNT=$COUNT ./create_config_maps.js
    done

    sleep 300

    for TEST in load_steve_k8s_pagination load_steve_new_pagination
    do
      for CLUSTER in local $DOWNSTREAM_CLUSTER
      do
          # warmup
          k6 run -e VUS=1 -e PER_VU_ITERATIONS=5 -e BASEURL=https://upstream.local.gd:8443 -e USERNAME=admin -e PASSWORD=adminadminadmin -e CLUSTER=$CLUSTER ./${TEST}.js

          # test + record
          k6 run -e VUS=10 -e PER_VU_ITERATIONS=30 -e BASEURL=https://upstream.local.gd:8443 -e USERNAME=admin -e PASSWORD=adminadminadmin -e CLUSTER=$CLUSTER ./${TEST}.js | tee ../docs/20230306\ -\ steve\ vai\ test\ results/${COUNT}_configmaps_${TEST}_${TAG}_image_${CLUSTER}_cluster_results.txt
          echo "**** DONE $COUNT - $TEST - $TAG - $CLUSTER"
      done
    done
  done
done
