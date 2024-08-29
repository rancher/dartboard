#!/bin/bash
set -e

oldPWD="$(pwd)"

dirs=("./scripts/soak" "./test" "./cmd/dartboard")

for dir in "${dirs[@]}"; do
    echo "Building $dir"
    cd "$dir"
    go build ./...
    cd "$oldPWD"
done
