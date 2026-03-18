#!/usr/bin/env bash
# Collect logs from all nodes deployed by dartboard
# Runs rancher2_logs_collector.sh on each node and downloads the resulting tarballs

set -euo pipefail

# Default values
CONFIG_DIR=""
OUTPUT_DIR="./collected_logs"
LOGS_COLLECTOR_URL="https://raw.githubusercontent.com/rancherlabs/support-tools/master/collection/rancher/v2.x/logs-collector/rancher2_logs_collector.sh"
PARALLEL_COUNT=5
DRY_RUN=false
VERBOSE=false
COLLECTOR_FLAGS=""

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Collect logs from all nodes deployed by dartboard using rancher2_logs_collector.sh

OPTIONS:
    -c, --config-dir DIR     Path to dartboard config directory (default: auto-detect from tofu workspace)
    -o, --output-dir DIR     Directory to store collected tarballs (default: ./collected_logs)
    -p, --parallel N         Number of nodes to process in parallel (default: 5)
    -f, --collector-flags    Additional flags to pass to rancher2_logs_collector.sh (e.g., "-s 3 -o")
    -n, --dry-run            Show what would be done without executing
    -v, --verbose            Enable verbose output
    -h, --help               Show this help message

EXAMPLES:
    # Auto-detect config directory and collect logs
    $(basename "$0")

    # Specify config directory explicitly
    $(basename "$0") -c ./tofu/main/aws/my-workspace_config

    # Collect logs with obfuscation enabled
    $(basename "$0") -f "-o"

    # Collect logs from last 3 days only
    $(basename "$0") -f "-s 3"

    # Run in dry-run mode to see what would happen
    $(basename "$0") -n

EOF
    exit 0
}

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        log "[VERBOSE] $*"
    fi
}

error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" >&2
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -c|--config-dir)
                CONFIG_DIR="$2"
                shift 2
                ;;
            -o|--output-dir)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            -p|--parallel)
                PARALLEL_COUNT="$2"
                shift 2
                ;;
            -f|--collector-flags)
                COLLECTOR_FLAGS="$2"
                shift 2
                ;;
            -n|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                usage
                ;;
            *)
                error "Unknown option: $1"
                usage
                ;;
        esac
    done
}

# Auto-detect the config directory from tofu workspace
auto_detect_config_dir() {
    local search_dirs=(
        "./tofu/main/aws"
        "./tofu/main/azure"
        "./tofu/main/harvester"
        "./tofu/main/k3d"
    )

    for dir in "${search_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            # Find directories ending with _config
            while IFS= read -r -d '' config_dir; do
                if [[ -f "$config_dir/clusters_state.yaml" ]]; then
                    echo "$config_dir"
                    return 0
                fi
            done < <(find "$dir" -maxdepth 1 -type d -name "*_config" -print0 2>/dev/null)
        fi
    done

    return 1
}

# Get all SSH scripts from the config directory
get_ssh_scripts() {
    local config_dir="$1"
    find "$config_dir" -maxdepth 1 -name "ssh-to-*.sh" -type f | sort
}

# Extract node name from SSH script filename
get_node_name() {
    local script="$1"
    basename "$script" .sh | sed 's/^ssh-to-//'
}

# Collect logs from a single node
collect_logs_from_node() {
    local ssh_script="$1"
    local node_name="$2"
    local output_dir="$3"
    local collector_flags="$4"

    log "[$node_name] Starting log collection..."

    # Commands to run on the remote node - collect logs and output tarball info
    # Use /var/tmp for storage to avoid issues with /tmp being read-only or tmpfs
    local remote_commands=$(cat <<REMOTE_SCRIPT
set -e

# Download the logs collector script
echo "Downloading logs collector..." >&2
sudo curl -sLo /var/tmp/rancher2_logs_collector.sh "$LOGS_COLLECTOR_URL"
sudo chmod +x /var/tmp/rancher2_logs_collector.sh

# Run the logs collector with /var/tmp as output directory
echo "Running logs collector..." >&2
sudo bash /var/tmp/rancher2_logs_collector.sh -d /var/tmp $collector_flags >&2 || true

# Find the most recently created tarball in /var/tmp
TARBALL=\$(sudo ls -t /var/tmp/*.tar.gz 2>/dev/null | head -1)

if [[ -n "\$TARBALL" && -f "\$TARBALL" ]]; then
    # Output tarball path with a unique marker for easy parsing
    echo "###TARBALL###\$TARBALL###"
else
    echo "ERROR: No tarball found in /var/tmp" >&2
    exit 1
fi
REMOTE_SCRIPT
)

    if [[ "$DRY_RUN" == "true" ]]; then
        log "[$node_name] [DRY-RUN] Would execute: $ssh_script"
        log "[$node_name] [DRY-RUN] Would download tarball to: $output_dir/"
        return 0
    fi

    # Execute on remote node and capture output
    local output
    if ! output=$("$ssh_script" bash -s <<< "$remote_commands" 2>&1); then
        error "[$node_name] Failed to collect logs"
        log "[$node_name] Output: $output"
        return 1
    fi

    # Show the collection output for visibility
    echo "$output" | grep -v "^###TARBALL###" >&2 || true

    # Extract tarball path using the marker
    local tarball_path
    tarball_path=$(echo "$output" | grep "^###TARBALL###" | sed 's/^###TARBALL###//;s/###$//' | tr -d '[:space:]')

    if [[ -z "$tarball_path" ]]; then
        error "[$node_name] Could not find tarball path in output"
        return 1
    fi

    # Validate the path looks like a tarball
    if [[ ! "$tarball_path" == *.tar.gz ]]; then
        error "[$node_name] Invalid tarball path: $tarball_path"
        return 1
    fi

    log "[$node_name] Remote tarball: $tarball_path"

    # Generate local filename
    local local_tarball="$output_dir/${node_name}-$(basename "$tarball_path")"

    log "[$node_name] Downloading tarball via SSH pipe..."

    # Download by piping through SSH - this reuses the existing SSH script
    # which already has all the correct proxy settings
    if "$ssh_script" "sudo cat '$tarball_path'" > "$local_tarball" 2>/dev/null; then
        # Verify the downloaded file is valid
        if [[ -s "$local_tarball" ]] && file "$local_tarball" | grep -q "gzip"; then
            log "[$node_name] Successfully downloaded: $local_tarball ($(du -h "$local_tarball" | cut -f1))"

            # Clean up remote tarball and script
            "$ssh_script" "sudo rm -f '$tarball_path' /var/tmp/rancher2_logs_collector.sh" 2>/dev/null || true

            return 0
        else
            error "[$node_name] Downloaded file is not a valid gzip archive"
            rm -f "$local_tarball"
            return 1
        fi
    else
        error "[$node_name] Failed to download tarball"
        rm -f "$local_tarball"
        return 1
    fi
}

# Process nodes with limited parallelism
process_nodes() {
    local ssh_scripts=("$@")
    local total=${#ssh_scripts[@]}
    local pids=()
    local node_names=()
    local results_dir

    results_dir=$(mktemp -d)

    log "Processing $total nodes with parallelism of $PARALLEL_COUNT..."

    for script in "${ssh_scripts[@]}"; do
        local node_name
        node_name=$(get_node_name "$script")

        # Wait if we've hit the parallel limit
        while [[ ${#pids[@]} -ge $PARALLEL_COUNT ]]; do
            # Wait for any child to complete
            local new_pids=()
            local new_names=()
            for i in "${!pids[@]}"; do
                if kill -0 "${pids[$i]}" 2>/dev/null; then
                    new_pids+=("${pids[$i]}")
                    new_names+=("${node_names[$i]}")
                else
                    # Process completed, wait for it to get exit code
                    wait "${pids[$i]}" 2>/dev/null || true
                fi
            done
            pids=("${new_pids[@]}")
            node_names=("${new_names[@]}")

            if [[ ${#pids[@]} -ge $PARALLEL_COUNT ]]; then
                sleep 1
            fi
        done

        # Start new collection in background
        log "Starting collection for $node_name ($(( ${#pids[@]} + 1 ))/$PARALLEL_COUNT active)..."
        (
            if collect_logs_from_node "$script" "$node_name" "$OUTPUT_DIR" "$COLLECTOR_FLAGS"; then
                touch "$results_dir/success-$node_name"
            else
                touch "$results_dir/failed-$node_name"
            fi
        ) &
        pids+=($!)
        node_names+=("$node_name")
    done

    # Wait for all remaining processes
    log "Waiting for remaining ${#pids[@]} processes to complete..."
    for pid in "${pids[@]}"; do
        wait "$pid" 2>/dev/null || true
    done

    # Count results
    local completed failed
    completed=$(find "$results_dir" -name "success-*" 2>/dev/null | wc -l)
    failed=$(find "$results_dir" -name "failed-*" 2>/dev/null | wc -l)

    rm -rf "$results_dir"

    log "Collection complete: $completed succeeded, $failed failed out of $total total"

    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    return 0
}

main() {
    parse_args "$@"

    # Auto-detect config directory if not specified
    if [[ -z "$CONFIG_DIR" ]]; then
        log "Auto-detecting config directory..."
        if CONFIG_DIR=$(auto_detect_config_dir); then
            log "Found config directory: $CONFIG_DIR"
        else
            error "Could not auto-detect config directory. Please specify with -c option."
            exit 1
        fi
    fi

    # Validate config directory
    if [[ ! -d "$CONFIG_DIR" ]]; then
        error "Config directory does not exist: $CONFIG_DIR"
        exit 1
    fi

    # Create output directory
    mkdir -p "$OUTPUT_DIR"
    log "Output directory: $OUTPUT_DIR"

    # Get SSH scripts
    mapfile -t ssh_scripts < <(get_ssh_scripts "$CONFIG_DIR")

    if [[ ${#ssh_scripts[@]} -eq 0 ]]; then
        error "No SSH scripts found in $CONFIG_DIR"
        exit 1
    fi

    log "Found ${#ssh_scripts[@]} nodes to collect logs from:"
    for script in "${ssh_scripts[@]}"; do
        log "  - $(get_node_name "$script")"
    done

    # Process all nodes
    if process_nodes "${ssh_scripts[@]}"; then
        log "All logs collected successfully!"
        log "Tarballs are stored in: $OUTPUT_DIR"
        ls -lah "$OUTPUT_DIR"/*.tar.gz 2>/dev/null || true
    else
        error "Some log collections failed. Check the output above for details."
        exit 1
    fi
}

main "$@"
