#!/usr/bin/env bash

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
SECONDS=0

#create parent output directory with date
parentdirdate=$(date '+%m-%d')
parentdir="soaktest-${parentdirdate}"

mkdir -p "${parentdir}"

#create output directory with timestamp
date=$(date '+%m-%d-%H-%M')
cr_dirname="cr-outputs-${date}"
first_cr_dir="${parentdir}/${cr_dirname}"
mkdir -p "${first_cr_dir}"
first_cr_dir=$(realpath "${first_cr_dir}")
echo "FIRST CR DIR: ${first_cr_dir}"

#run cluster resource count script on clusters from provided kubeconfigs
#usage: parallel -k --lb ./cr.sh ::: kubeconfig1.yaml kubeconfig2.yaml kubeconfig3.yaml kubeconfig4.yaml kubeconfig5.yaml
find "${script_dir}" -maxdepth 1 -type f -name '*.yaml' -print0 | parallel -P 7 -0 -k --group ./cr.sh {} ${first_cr_dir}

wait

parentdir=$(dirname "${first_cr_dir}")

#create timestamped directory for results
date=$(date '+%m-%d-%H-%M')
soak_dirname="soaktest-${date}"
soak_dir="${parentdir}/$soak_dirname"
mkdir -p "${soak_dir}"
echo "SOAK DIR: ${soak_dir}"

#run soak test on clusters from provided kubeconfigs
#Usage: parallel -k --lb --res $soak_dirname serve_hostnames ::: kubeconfig1.yaml kubeconfig2.yaml kubeconfig3.yaml kubeconfig4.yaml kubeconfig5.yaml
# parallel -P 7 -v -k --group --res "${soak_dir}" serve_hostnames -kubeconfig={1} -up_to=-1 -pods_per_node=4 -max_dur=3 ::: rke2-1-26-8-custom.yaml local.yaml k3s-1-26-8-psa.yaml rke1-1-26-8-custom.yaml rke2-1-26-8-psa.yaml rke1-1-26-8-psa.yaml k3s-1-26-8-custom.yaml
# find . -maxdepth 1 -name "*.yaml" | parallel -v -k --group --res "${soak_dir}" "./serve_hostnames {}"
# parallel -v -k --group --res "${soak_dir}" serve_hostnames -kubeconfig={1} -up_to=-1 -pods_per_node=4 -max_dur=30 ::: local.yaml k3s-custom.yaml rke1-custom.yaml rke2-custom.yaml k3s-1-26-7-psa.yaml rke1-1-26-7-psa.yaml rke2-1-26-7-psa.yaml
find . -maxdepth 1 -type f -name '*.yaml' -print0 | parallel -P 7 -0 -v -k --group --res "${soak_dir}" serve_hostnames -kubeconfig={} -up_to=-1 -pods_per_node=4 -max_dur=30
wait

#create output directory with timestamp
date=$(date '+%m-%d-%H-%M')
cr_dirname="cr-outputs-${date}"
second_cr_dir="${parentdir}/${cr_dirname}"
mkdir -p "${second_cr_dir}"
second_cr_dir=$(realpath "${second_cr_dir}")
echo "SECOND CR DIR: ${second_cr_dir}"

#run cluster resource count script on clusters from provided kubeconfigs
#usage: parallel -k --lb ./cr.sh ::: kubeconfig1.yaml kubeconfig2.yaml kubeconfig3.yaml kubeconfig4.yaml kubeconfig5.yaml
find "${script_dir}" -maxdepth 1 -type f -name '*.yaml' -print0 | parallel -P 7 -0 -k --group ./cr.sh {} ${second_cr_dir}

wait

duration=$SECONDS
echo "$((duration / 60)) minutes and $((duration % 60)) seconds elapsed."

date=$(date '+%m-%d-%H-%M')
diff -rub "${first_cr_dir}" "${second_cr_dir}" >"${parentdir}/cr_diff-${date}.txt"
