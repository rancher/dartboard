#!/usr/bin/env bash

NAMESPACE=default

FIRST_CLUSTER_NAME=$(kubectl get --namespace=$NAMESPACE cluster --output=custom-columns=":metadata.name" | grep cluster- | head -1)
kubectl get --namespace=$NAMESPACE cluster $FIRST_CLUSTER_NAME --output=yaml > cluster.yaml

for i in {1..2000}
do
  CLUSTER_ID=`echo $i | md5sum | head --bytes 12`
  cat cluster.yaml | sed "s/$FIRST_CLUSTER_NAME/cluster-$CLUSTER_ID/g" | kubectl apply --filename=-
done
