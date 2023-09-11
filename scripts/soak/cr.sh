#!/usr/bin/env bash

#Usage: ./cr.sh kubeconfig.yaml

#set the path for the kubeconfig
export KUBECONFIG=$1

#create parent output directory with date
parentdirdate=$(date '+%m-%d')
parentdir=soaktest-$parentdirdate

mkdir -p "$parentdir"

#create output directory with timestamp
date=$(date '+%m-%d-%H-%M')
dirname=cr-outputs-$date
mkdir -p "$parentdir/$dirname"

#create output filename with timestamp from kubeconfig filename
kubeconfigname=$1
kubeconfigname=${kubeconfigname%.*}
date=$(date '+%m-%d-%H-%M-%S')
filename=$kubeconfigname-$date".txt"

#kubectl api-resources call lists all resources, loop on each resource
for resource in $(kubectl api-resources -o wide | grep -v "NAME" | awk '{ print $1 }'); do
  #output the resource name to file
  echo -n " ${resource} : " >>"$parentdir/$dirname/$filename"
  #kubeclt get call on each resource for all namespaces, loop and count lines ignoring column title lines
  for count in $(kubectl get "$resource" -A | grep -v "NAMESPACE" | grep -v "NAME" | wc -l); do
    #output to file
    echo "${count}" >>"$parentdir/$dirname/$filename"
  done
done
