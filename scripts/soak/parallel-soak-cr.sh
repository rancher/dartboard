#!/usr/bin/env bash

SECONDS=0

#run cluster resoruce count script on clusters from provided kubeconfigs
#usage: parallel -k --lb ./cr.sh ::: kubeconfig1.yaml kubeconfig2.yaml kubeconfig3.yaml kubeconfig4.yaml kubeconfig5.yaml
find . -maxdepth 1 -type f -name '*.yaml' -print0 | parallel -0 -k --group ./cr.sh {}

wait

parentdirdate=$(date '+%m-%d')
parentdir=soaktest-$parentdirdate

mkdir -p "$parentdir"

#create timestamped directory for results
date=$(date '+%m-%d-%H-%M')
dirname=soaktest-$date
mkdir -p "$parentdir/$dirname"

#run soak test on clusters from provided kubeconfigs
#Usage: parallel -k --lb --res $dirname serve_hostnames ::: kubeconfig1.yaml kubeconfig2.yaml kubeconfig3.yaml kubeconfig4.yaml kubeconfig5.yaml
# parallel -v -k --group --res "$parentdir/$dirname" serve_hostnames -up_to=-1 -pods_per_node=4 ::: k3s-custom.yaml rke1-custom.yaml rke2-custom.yaml k3s-1-26-7-psa.yaml rke1-1-26-7-psa.yaml rke2-1-26-7-psa.yaml
# find . -maxdepth 1 -name "*.yaml" | parallel -v -k --group --res "$parentdir/$dirname" "./serve_hostnames {}"
# parallel -v -k --group --res "$parentdir/$dirname" serve_hostnames -kubeconfig={1} -up_to=-1 -pods_per_node=4 -max_dur=30 ::: local.yaml k3s-custom.yaml rke1-custom.yaml rke2-custom.yaml k3s-1-26-7-psa.yaml rke1-1-26-7-psa.yaml rke2-1-26-7-psa.yaml
find . -maxdepth 1 -type f -name '*.yaml' -print0 | parallel -0 -v -k --group --res "$parentdir/$dirname" serve_hostnames -kubeconfig={} -up_to=-1 -pods_per_node=4 -max_dur=30
wait

#run cluster resoruce count script on clusters from provided kubeconfigs
#usage: parallel -k --lb ./cr.sh ::: kubeconfig1.yaml kubeconfig2.yaml kubeconfig3.yaml kubeconfig4.yaml kubeconfig5.yaml
find . -maxdepth 1 -type f -name '*.yaml' -print0 | parallel -0 -k --group ./cr.sh {}

wait

duration=$SECONDS
echo "$((duration / 60)) minutes and $((duration % 60)) seconds elapsed."
