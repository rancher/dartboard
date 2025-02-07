#!/usr/bin/env bash

# set kubeconfig
export KUBECONFIG=$1

# set timestamp, create dir for export path, set permissions, navigate
st1=$(date +"%Y-%m-%d")
mkdir -p $PWD/metrics-$st1
chmod +x metrics-$st1
cd metrics-$st1

selector='{__name__!=""}' 


# run script for mimirtool via kubectl on target cluster
bash -c "./../import-metrics-mimirtool.sh $1" &

#TODO: improve wait
sleep 5


for i in 6 4 2
do

j=$((${i}-2))


from="$(date -u -v-${i}H +"%Y-%m-%dT%H:%M:00Z")"
to="$(date -u -v-${j}H +"%Y-%m-%dT%H:%M:00Z")"

# from separate mimirtool shell execute remote read
kubectl exec -n cattle-monitoring-system mimirtool --insecure-skip-tls-verify -i -t -- mimirtool remote-read export --tsdb-path ./prometheus-export --address http://rancher-monitoring-prometheus:9090 --remote-read-path /api/v1/read --to=$to --from=$from --selector $selector

# compress metrics data from export
kubectl exec -n cattle-monitoring-system mimirtool --insecure-skip-tls-verify -i -t -- tar zcf /tmp/prometheus-export.tar.gz ./prometheus-export

# set filename timestamp
st2=$(date +"%Y-%m-%d--%H-%M-%S")

# copy exported metrics data to timestamped tarball
kubectl -n cattle-monitoring-system cp mimirtool:/tmp/prometheus-export.tar.gz ./prometheus-export-$st2.tar.gz

# clear export data from pod, 
#kubectl exec -n cattle-monitoring-system mimirtool --insecure-skip-tls-verify -i -t -- rm -rf prometheus-export

# unpack, navigate
tar xf prometheus-export-$st2.tar.gz
cd prometheus-export

# aggregate tsdb
cp -R `ls | grep -v "wal"` ../

# cleanup
cd ../
rm -rf prometheus-export

sleep 5


done

# kill mimirtool pod
kubectl delete pod -n cattle-monitoring-system mimirtool

# run prometheus graph on metrics data (locally via docker), overlapping/obsolete blocks are deleted during compaction
docker run --rm -u "$(id -u)" -ti -p 9090:9090 -v $PWD:/prometheus rancher/mirrored-prometheus-prometheus:v2.42.0 --storage.tsdb.path=/prometheus --storage.tsdb.retention.time=1y --config.file=/dev/null

