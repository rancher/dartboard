#!/bin/bash

set -x
set -eu

DEBUG="${DEBUG:-false}"

env | grep -E '^(AWS|RKE2).*\=.+' | sort > .env

if [ "false" != "${DEBUG}" ]; then
    cat .env
fi