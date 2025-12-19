#!/usr/bin/env bash
set -e

# Path to the Go source code for the qase-k6-cli
QASE_K6_CLI_SRC_DIR="$(dirname "$0")"

# Output binary path
OUTPUT_BINARY="${QASE_K6_CLI_SRC_DIR}/qase-k6-cli"

echo "Building qase-k6-cli..."
echo "Source directory: ${QASE_K6_CLI_SRC_DIR}"
echo "Output binary: ${OUTPUT_BINARY}"

# Ensure we are in the correct directory to resolve modules
cd "${QASE_K6_CLI_SRC_DIR}"

# Tidy and build the Go application
go mod tidy
go build -o "${OUTPUT_BINARY}" .

echo "Build complete."
