#!/usr/bin/env bash

# Script using mimirtool for prometheus tsdb collection
#
# Usage: ./import-metrics.sh path/to/kubeconfig.yaml
#
# Important: Must set inputs for selector and time range
# 
# For all metrics use selector='{__name__!=""}'
# Time format YYYY-MM-DDThh:mm:ssZ  ex: "2025-01-31T00:00:00Z"
# bash helpers: 
#      date -u +"%Y-%m-%dT%H:%M:%SZ"
#      date -u -v-3H +"%Y-%m-%dT%H:%M:%SZ"
#      [-v[+|-]val[y|m|w|d|H|M|S]]
selector='{__name__!=""}' 
from="$(date -u -v-2H +"%Y-%m-%dT%H:%M:%SZ")"
to="$(date -u -v-0H +"%Y-%m-%dT%H:%M:%SZ")"

# set kubeconfig
export KUBECONFIG=$1

#TODO: confirm access to cluster
#TODO: check and increase memory for prometheus installation

# set timestamp, create dir for export path, set permissions, navigate
st1=$(date +"%Y-%m-%d")
mkdir -p $PWD/metrics-$st1
chmod +x metrics-$st1
cd metrics-$st1

# clean any previous mimirtool pod
#kubectl delete pod -n cattle-monitoring-system mimirtool

# run script for mimirtool via kubectl on target cluster
bash -c "./../import-metrics-mimirtool.sh $1" &

#TODO: improve wait
sleep 5

# from separate mimirtool shell execute remote read
kubectl exec -n cattle-monitoring-system mimirtool --insecure-skip-tls-verify -i -t -- mimirtool remote-read export --tsdb-path ./prometheus-export --address http://rancher-monitoring-prometheus:9090 --remote-read-path /api/v1/read --to=$to --from=$from --selector $selector

# compress metrics data from export
kubectl exec -n cattle-monitoring-system mimirtool --insecure-skip-tls-verify -i -t -- tar zcf /tmp/prometheus-export.tar.gz ./prometheus-export

# set filename timestamp
st2=$(date +"%Y-%m-%d--%H-%M")

# copy exported metrics data to timestamped tarball
kubectl -n cattle-monitoring-system cp mimirtool:/tmp/prometheus-export.tar.gz ./prometheus-export-${st2}.tar.gz

# unpack, navigate
tar xf prometheus-export-$st2.tar.gz
cd prometheus-export

# aggregate tsdb
cp -R `ls | grep -v "wal"` ../

# cleanup
cd ../
rm -rf prometheus-export

# kill mimirtool pod
kubectl delete pod -n cattle-monitoring-system mimirtool

# run prometheus graph on metrics data (locally via docker), overlapping/obsolete blocks are deleted during compaction
docker run --rm -u "$(id -u)" -ti -p 9090:9090 -v $PWD:/prometheus rancher/mirrored-prometheus-prometheus:v2.42.0 --storage.tsdb.path=/prometheus --storage.tsdb.retention.time=1y --config.file=/dev/null

