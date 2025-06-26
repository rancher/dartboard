#!/usr/bin/env bash

set -o errexit
set -o pipefail

# Script using mimirtool for prometheus tsdb collection
# 
# Usage: ./export-metrics.sh /path/to/kubeconfig.yaml selector from to offset
#
#   arg: path to kubeconfig (required via cli or as environment variable)
#   arg: prometheus query selector (optional)
#   arg: target date for query to start from (optional)
#   arg: target date for query to end (optional)
#   arg: offset(in seconds) (optional)
#
# See README for more usage information

# offset_seconds - query time range loop increment, only modify if default prometheus installation memory has been increased, default is set for one hour
offset_seconds=3600 # one hour

# selector - a valid prometheus query in single quotes, default selector set for ALL METRICS
selector='{__name__!=""}'

# determine os for date commands
os_uname=$(uname)

main() { 

    # from - date for query to begin
    # to - date for query to end
    # default time range set for ONE HOUR from current utc time
    from="$(get_date 1)" 
    to="$(get_date)"

    # convert default dates for comparisons for macOS
    to_seconds="$(date_to_seconds "${to}")"
    from_seconds="$(date_to_seconds "${from}")"

    # set input variables from script arguments
    process_args "$@"

    # display parameters for metrics export
    printf "Starting export-metrics script...\n\n Prometheus query: %s  \n Query start: %s \n Query end:  %s" "${selector}"  "${from}" "${to}"

    # only display offset if default has been changed
    if [ "$offset_seconds" -gt 3600 ]; then
        printf "\n OFFSET: %s \n\n" "${offset_seconds}"
    else
        printf "\n\n"
    fi

    # confirm access to cluster or exit
    kubectl get all -A 1> /dev/null || exit 1
    printf " - Confirm kubeconfig access \e[32mPASS\e[0m \n"

    # clean any previous mimirtool pod
    kubectl delete pod -n cattle-monitoring-system mimirtool || printf " - Check for prior mimirtool instance \e[32mPASS\e[0m \n"

    # run mimirtool pod on target cluster
    kubectl apply -f "${PWD}"/mimirtool.yaml

    # wait for mimirtool pod to start
    sleep 10

    # confirm mimirtool pod is running
    kubectl exec -n cattle-monitoring-system mimirtool --insecure-skip-tls-verify -i -t -- ls 1> /dev/null || exit 1
    printf " - Confirm mimirtool pod is running \e[32mPASS\e[0m \n"

    # set timestamp, create dir for export path, set permissions, navigate
    ts1=$(date +"%Y-%m-%d")
    kube_name=$(printf "%s" "${KUBECONFIG##*/}"  | cut -d '.' -f1)
    mkdir -p "${PWD}"/metrics-"$kube_name"-"$ts1"
    chmod +x metrics-"$kube_name"-"$ts1"
    cd metrics-"$kube_name"-"$ts1"

    # iterate queries on offset_seconds in reverse backwards in time from target "to" date to target "from" date
    while [ "${to_seconds}" -gt "${from_seconds}" ]; do

        # reduce offset_seconds when last query time range will be less than offset
        if [ $((to_seconds - from_seconds)) -lt "${offset_seconds}" ]; then
            offset_seconds=$((to_seconds - from_seconds))
        fi

        # set date range for query
        range=$((to_seconds - offset_seconds))

        from="$(seconds_to_date "${range}")"
        to="$(seconds_to_date "${to_seconds}")"

        # from separate mimirtool shell execute remote-read
        kubectl exec -n cattle-monitoring-system mimirtool --insecure-skip-tls-verify -i -t -- mimirtool remote-read export --tsdb-path ./prometheus-export --address http://rancher-monitoring-prometheus:9090 --remote-read-path /api/v1/read --to="${to}" --from="${from}" --selector "${selector}"

        # compress metrics data from export
        kubectl exec -n cattle-monitoring-system mimirtool --insecure-skip-tls-verify -i -t -- tar zcf /tmp/prometheus-export.tar.gz ./prometheus-export

        # set filename timestamp
        ts2="$(get_timestamp "${range}")"

        # copy exported metrics data to timestamped tarball
        kubectl -n cattle-monitoring-system cp mimirtool:/tmp/prometheus-export.tar.gz ./prometheus-export-"${ts2}".tar.gz 1> /dev/null

        # clear export data from pod 
        kubectl exec -n cattle-monitoring-system mimirtool --insecure-skip-tls-verify -i -t -- rm -rf prometheus-export

        # unpack, navigate, clean
        tar xf prometheus-export-"${ts2}".tar.gz 1> /dev/null
        cd prometheus-export
        rm -r wal

        # aggregate tsdb
        tsdb_count=$(find "$PWD" -type d -not -path '*/.*' -mindepth 1 | wc -l)
        if [ "$tsdb_count" -eq 0 ]; then
            printf " - No blocks to copy \n"
            rm ../prometheus-export-"${ts2}".tar.gz
        else
            cp -R "${PWD}"/* ../
        fi

        # navigate, cleanup
        cd ../
        rm -r prometheus-export

        # increment time range by offset_seconds
        to_seconds=$((to_seconds - offset_seconds))

        # wait
        sleep 5
        
    done

    # delete mimirtool pod
    kubectl delete pod -n cattle-monitoring-system mimirtool

    # output command to run prometheus graph on metrics data (locally via docker, overlapping/obsolete blocks are handled during compaction)
    printf "\n\e[32mMetrics export complete!\e[0m\nCopy and/or view metrics data locally:\n\n"
    printf "scp -r -i path/for/key user@address:/path/on/remote/metrics-\* /path/for/local/ \n\n"
    printf "docker run --rm -u %s -ti -p 9090:9090 -v ${PWD}:/prometheus rancher/mirrored-prometheus-prometheus:v2.42.0 --storage.tsdb.path=/prometheus --storage.tsdb.retention.time=1y --config.file=/dev/null \n\n" "$(id -u)"

}

# execute date command based on os, use default of 1 hour ago if any argument is present
get_date(){

    if [ "$os_uname" = "Darwin" ]; then 
        if [ $# -eq 1 ]; then
            date -u -v-1H +"%Y-%m-%dT%H:%M:%SZ"
        else
            date -u +"%Y-%m-%dT%H:%M:%SZ"
        fi
    elif [ "$os_uname" = "Linux" ]; then
        if [ $# -eq 1 ]; then
            date -u --date="1 hour ago" +"%Y-%m-%dT%H:%M:%SZ"
        else
            date -u +"%Y-%m-%dT%H:%M:%SZ"
        fi
    fi

}

# convert default dates to seconds for comparisons
date_to_seconds(){

    if [ "$os_uname" = "Darwin" ]; then 
        date -j -f "%Y-%m-%dT%H:%M:%SZ" "$1" "+%s"
    elif [ "$os_uname" = "Linux" ]; then
        date -d "$1" "+%s"
    fi

}

# convert seconds to default date format
seconds_to_date(){

    if [ "$os_uname" = "Darwin" ]; then 
        date -j -f "%s" "$1" "+%Y-%m-%dT%H:%M:%SZ"
    elif [ "$os_uname" = "Linux" ]; then
        date -d @"$1" "+%Y-%m-%dT%H:%M:%SZ"
    fi

}

# execute date command with timestamp format
get_timestamp(){

    if [ "$os_uname" = "Darwin" ]; then 
        date -j -f "%s" "$1" "+%Y-%m-%dT%H-%M-%S"
    elif [ "$os_uname" = "Linux" ]; then
        date -d @"$1" "+%Y-%m-%dT%H-%M-%S"
    fi

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
            export KUBECONFIG=$arg
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
                temp_seconds="$(date_to_seconds "$arg")"
                if [ "$temp_seconds" -lt "$from_seconds" ]; then
                    from=$arg
                    date_count=$((date_count+1))
                fi
            fi        
        fi
        
    done

    # limit offset to two hours
    if [ "${offset_seconds}" -gt 7200 ]; then
        offset_seconds=7200

        # check prometheus memory, limit offset to 1hr if <= 3000Mi
        prometheus_memory=$(kubectl get statefulsets -n cattle-monitoring-system prometheus-rancher-monitoring-prometheus -o jsonpath='{.spec.template.spec.containers[0].resources.limits.memory}' | tr -d "Mi")
        if [ "${prometheus_memory}" -lt 3001 ]; then
            offset_seconds=3600
        fi
    fi

    # overwrite defaults and convert input dates for comparisons
    to_seconds="$(date_to_seconds "${to}")"
    from_seconds="$(date_to_seconds "${from}")"

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
