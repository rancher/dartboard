#!/usr/bin/env bash
set -euo pipefail
[[ "${DEBUG:-}" == "1" ]] && set -x

# Path to the Go source code for the resource-counts tool
CR_SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Output binary path (place binary next to this script)
OUTPUT_BINARY="${CR_SRC_DIR}/count-resources"

echo "Building count-resources..."
echo "Source directory: ${CR_SRC_DIR}"
echo "Output binary: ${OUTPUT_BINARY}"

# Ensure we are in the correct directory to resolve modules
cd "${CR_SRC_DIR}"

# Tidy and build the Go application
go mod tidy
go build -o "${OUTPUT_BINARY}" .

echo "Build complete."
