#!/usr/bin/env bash
set -e

# Path to the Go source code for the k6 reporter
REPORTER_SRC_DIR="$(dirname "$0")"

# Output binary path
OUTPUT_BINARY="${REPORTER_SRC_DIR}/reporter-k6"

echo "Building k6 Qase reporter..."
echo "Source directory: ${REPORTER_SRC_DIR}"
echo "Output binary: ${OUTPUT_BINARY}"

# Ensure we are in the correct directory to resolve modules
cd "${REPORTER_SRC_DIR}"

# Tidy and build the Go application
go mod tidy
go build -o "${OUTPUT_BINARY}" .

echo "Build complete."
