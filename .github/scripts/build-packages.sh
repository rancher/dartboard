#!/bin/bash
set -e

oldPWD="$(pwd)"

dirs=("./scripts/soak" "./utils" "./test")

for dir in "${dirs[@]}"; do
    echo "Building $dir"
    cd "$dir"
    go build ./...
    cd "$oldPWD"
done
