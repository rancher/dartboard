#!/usr/bin/env bash

#Usage: ./cr.sh kubeconfig.yaml

#set the path for the kubeconfig
export KUBECONFIG=$1

#create output filename with timestamp from kubeconfig filename
kubeconfigname="${1}"
cr_dir="${2}"
kubeconfigname="${kubeconfigname%.*}"
kubeconfig_filename="$(basename ${kubeconfigname}.txt)"

start_date=$(date '+%m-%d-%H-%M-%S')
touch "${cr_dir}/start-${start_date}.txt"

#kubectl api-resources call lists all resources, loop on each resource
for resource in $(kubectl api-resources -o wide | grep -v "NAME" | awk '{ print $1 }'); do
  # output the resource name to file
  echo -n " ${resource} : " >>"${cr_dir}/${kubeconfig_filename}"
  # kubectl get call on each resource for all namespaces, loop and count lines ignoring column title lines
  for count in $(kubectl get "$resource" -A | grep -v "NAMESPACE" | grep -v "NAME" | wc -l); do
    # output to file
    echo "${count}" >>"${cr_dir}/${kubeconfig_filename}"
  done
done

end_date=$(date '+%m-%d-%H-%M-%S')
touch "${cr_dir}/end-${end_date}.txt"
