#!/usr/bin/env bash

set -o errexit
set -o pipefail

# Script using mimirtool for prometheus tsdb collection
# 
# Usage: ./import-metrics.sh /path/to/kubeconfig.yaml selector from to offset
#
#   arg: path to kubeconfig (required via cli or as environment variable)
#   arg: prometheus query selector (optional)
#   arg: target date for query to start from (optional)
#   arg: target date for query to end (optional)
#   arg: offset(in seconds (optional)
#
# See README for more usage information

# offset_seconds - query time range loop increment, only modify if default prometheus installation memory has been increased, default is set for one hour
offset_seconds=3600 # one hour

# selector - a valid prometheus query in single quotes, default selector set for ALL METRICS
selector='{__name__!=""}'

# from - date for query to begin
# to - date for query to end
# default time range set for ONE HOUR from current utc time
from="$(date -u -v-1H +"%Y-%m-%dT%H:%M:%SZ")" 
to="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

# convert default dates for comparisons
to_seconds=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "${to}" "+%s")
from_seconds=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "${from}" "+%s")

main() { 

    # set input variables from script arguments
    process_args "$@"

    # display parameters for metrics import
    printf "Starting import-metrics script...\n\n Prometheus query: ${selector}  \n Query start: ${from} \n Query end:  ${to}"

    # only display offset if default has been changed
    if [ $offset_seconds -gt 3600 ]; then
        printf "\n OFFSET: ${offset_seconds} \n\n" 
    else
        printf "\n\n"
    fi

    # confirm access to cluster or exit
    kubectl get all -A 1> /dev/null || exit 1
    printf " - Confirm kubeconfig access \e[32mPASS\e[0m \n"

    # clean any previous mimirtool pod
    kubectl delete pod -n cattle-monitoring-system mimirtool || printf " - Check for prior mimirtool instance \e[32mPASS\e[0m \n"

    # run mimirtool pod on target cluster
    kubectl apply -f ${PWD}/mimirtool.yaml

    # wait for mimirtool pod to start
    sleep 5

    # confirm mimirtool pod is running
    kubectl exec -n cattle-monitoring-system mimirtool --insecure-skip-tls-verify -i -t -- ls 1> /dev/null || exit 1
    printf " - Confirm mimirtool pod is running \e[32mPASS\e[0m \n"

    # set timestamp, create dir for export path, set permissions, navigate
    st1=$(date +"%Y-%m-%d")
    mkdir -p ${PWD}/metrics-$st1
    chmod +x metrics-$st1
    cd metrics-$st1

    # iterate queries on offset_seconds in reverse backwards in time from target "to" date to target "from" date
    while [ "${to_seconds}" -gt "${from_seconds}" ]; do

        # reduce offset_seconds when last query time range will be less than offset
        if [ $((${to_seconds} - ${from_seconds})) -lt ${offset_seconds} ]; then
            offset_seconds=$((${to_seconds} - ${from_seconds}))
        fi

        # set date range for query
        range=$((${to_seconds} - ${offset_seconds}))
        from=$(date -j -f "%s" "${range}" "+%Y-%m-%dT%H:%M:%SZ")
        to=$(date -j -f "%s" "${to_seconds}" "+%Y-%m-%dT%H:%M:%SZ")

        # from separate mimirtool shell execute remote-read
        kubectl exec -n cattle-monitoring-system mimirtool --insecure-skip-tls-verify -i -t -- mimirtool remote-read export --tsdb-path ./prometheus-export --address http://rancher-monitoring-prometheus:9090 --remote-read-path /api/v1/read --to=${to} --from=${from} --selector ${selector}

        # compress metrics data from export
        kubectl exec -n cattle-monitoring-system mimirtool --insecure-skip-tls-verify -i -t -- tar zcf /tmp/prometheus-export.tar.gz ./prometheus-export

        # set filename timestamp
        st2=$(date -j -f "%s" "${range}" "+%Y-%m-%dT%H-%M-%S")

        # copy exported metrics data to timestamped tarball
        kubectl -n cattle-monitoring-system cp mimirtool:/tmp/prometheus-export.tar.gz ./prometheus-export-${st2}.tar.gz 1> /dev/null

        # clear export data from pod 
        kubectl exec -n cattle-monitoring-system mimirtool --insecure-skip-tls-verify -i -t -- rm -rf prometheus-export

        # unpack, navigate
        tar xf prometheus-export-${st2}.tar.gz 1> /dev/null
        cd prometheus-export

        # aggregate tsdb
        cp -R `ls | grep -v "wal"` ../ || printf " - No blocks to copy \n"
        
        # cleanup
        cd ../
        rm -rf prometheus-export

        # increment time range by offset_seconds
        to_seconds=$((${to_seconds} - ${offset_seconds}))

        # wait
        sleep 5
        
    done

    # delete mimirtool pod
    kubectl delete pod -n cattle-monitoring-system mimirtool

    # output command to run prometheus graph on metrics data (locally via docker, overlapping/obsolete blocks are handled during compaction)
    printf "\n\e[32mMetrics import complete!\e[0m\nView metrics data locally via docker:\n\n"
    printf "docker run --rm -u \"$(id -u)\" -ti -p 9090:9090 -v ${PWD}:/prometheus rancher/mirrored-prometheus-prometheus:v2.42.0 --storage.tsdb.path=/prometheus --storage.tsdb.retention.time=1y --config.file=/dev/null \n\n"

}


# set input variables from script arguments
process_args(){

    # regex to match arguments
    kube_regex="(.*yaml)|(.*yml)"
    selector_regex="({.*})"
    offset_regex="[0-9]{4}(\-){0}$"
    date_regex=".*T.*Z"

    # count date inputs
    date_count=0
    
    for arg in "$@"
    do
        # set kubeconfig from input
        if [[ $arg =~ ${kube_regex} ]]; then
            export KUBECONFIG=$1
        fi

        # set prometheus query from input
        if [[ $arg =~ ${selector_regex} ]]; then
            selector=$arg
        fi

        # check and set offset_seconds
        if [[ $arg =~ ${offset_regex} ]]; then
            offset_seconds=$arg
        fi

        # check and set dates
        if [[ $arg =~ ${date_regex} ]]; then
            if [ $date_count = 1 ]; then
                to=$arg
            fi
        
            if [ $date_count = 0 ]; then
                temp_seconds=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "${arg}" "+%s")
                if [ $temp_seconds -lt $from_seconds ]; then
                    from=$arg
                    date_count=$((date_count+1))
                fi
            fi        
        fi
        
    done

    # limit offset to two hours
    if [ "${offset_seconds}" -gt 7200 ]; then
        offset_seconds=7200
        # TODO: Check prometheus installation memory, limit to 3600 if <=3GB
    fi

    # overwrite defaults and convert input dates for comparisons
    to_seconds=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "${to}" "+%s")
    from_seconds=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "${from}" "+%s")

    # check dates and ensure TO and FROM are set appropriately 
    if [ "${to_seconds}" -lt "${from_seconds}" ]; then
        from_temp="${from_seconds}"
        from_seconds="${to_seconds}"
        to_seconds="${from_temp}"

        from_temp="${from}"
        from="${to}"
        to="${from_temp}"
    fi

}

main "$@"
