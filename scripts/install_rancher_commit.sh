#!/usr/bin/env bash
set -euo pipefail

COMMIT=${1#"v"}
if [ -z "$COMMIT" ]; then
  echo "Must specify version!"
  exit 1
fi
VERSION=$(
  curl -sS "https://proxy.golang.org/github.com/rancher/rancher/@v/${COMMIT}.info" |
    jq -r .Version
)
go install "github.com/rancher/rancher@${VERSION}"
