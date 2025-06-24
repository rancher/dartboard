#!/bin/bash

set -x
set -eu

DEBUG="${DEBUG:-false}"

TRIM_JOB_NAME=$(basename "$JOB_NAME")

if [ "false" != "${DEBUG}" ]; then
    echo "Environment:"
    env | sort
fi

count=0
while [[ 3 -gt $count ]]; do
    docker build . -f scripts/Dockerfile.k6 -t k6-dartboard-"${TRIM_JOB_NAME}""${BUILD_NUMBER}"

    BUILD_EXIT_CODE=$?
    if [[ $BUILD_EXIT_CODE -eq 0 ]]; then break; fi
    count=$((count + 1))
    echo "Repeating failed Docker build ${count} of 3..."
done