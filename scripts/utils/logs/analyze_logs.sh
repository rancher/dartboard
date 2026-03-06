#!/usr/bin/env bash
# Analyze collected logs for errors in key Rancher components
# Focuses on: cattle-agent, CAPI, system-upgrade-controller, rancher pods

set -euo pipefail

# Default values
LOGS_DIR="./collected_logs"
OUTPUT_FILE=""
VERBOSE=false
INCLUDE_WARNINGS=false
CONTEXT_LINES=3
MAX_ERRORS_PER_COMPONENT=50

# Components to analyze
COMPONENTS=(
    "cattle-agent"
    "cattle-cluster-agent"
    "rancher"
    "system-upgrade-controller"
    "capi"
    "cluster-api"
    "fleet"
    "webhook"
)

# Error patterns to search for
ERROR_PATTERNS=(
    "error"
    "Error"
    "ERROR"
    "fatal"
    "Fatal"
    "FATAL"
    "panic"
    "Panic"
    "PANIC"
    "failed"
    "Failed"
    "FAILED"
    "exception"
    "Exception"
    "timeout"
    "Timeout"
    "refused"
    "Refused"
    "unavailable"
    "Unavailable"
    "OOMKilled"
    "CrashLoopBackOff"
    "ImagePullBackOff"
    "ErrImagePull"
    "BackOff"
    "evicted"
    "Evicted"
)

WARNING_PATTERNS=(
    "warn"
    "Warn"
    "WARN"
    "warning"
    "Warning"
    "WARNING"
    "deprecated"
    "Deprecated"
)

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Analyze collected dartboard logs for errors in key Rancher components.

OPTIONS:
    -d, --logs-dir DIR       Directory containing collected log tarballs (default: ./collected-logs)
    -o, --output FILE        Write report to file (default: stdout)
    -w, --warnings           Include warnings in addition to errors
    -c, --context N          Number of context lines around errors (default: 3)
    -m, --max-errors N       Max errors to show per component (default: 50)
    -v, --verbose            Show verbose output during analysis
    -h, --help               Show this help message

ANALYZED COMPONENTS:
    - cattle-agent / cattle-cluster-agent
    - rancher (server pods)
    - system-upgrade-controller
    - CAPI (Cluster API)
    - fleet
    - webhooks

EXAMPLES:
    # Analyze logs in default directory
    $(basename "$0")

    # Analyze with warnings and save to file
    $(basename "$0") -w -o analysis-report.txt

    # Analyze specific directory with more context
    $(basename "$0") -d ./my-logs -c 5

EOF
    exit 0
}

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >&2
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        log "[VERBOSE] $*"
    fi
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -d|--logs-dir)
                LOGS_DIR="$2"
                shift 2
                ;;
            -o|--output)
                OUTPUT_FILE="$2"
                shift 2
                ;;
            -w|--warnings)
                INCLUDE_WARNINGS=true
                shift
                ;;
            -c|--context)
                CONTEXT_LINES="$2"
                shift 2
                ;;
            -m|--max-errors)
                MAX_ERRORS_PER_COMPONENT="$2"
                shift 2
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                usage
                ;;
            *)
                echo "Unknown option: $1" >&2
                usage
                ;;
        esac
    done
}

# Build grep pattern from array
build_pattern() {
    local -n patterns=$1
    local result=""
    for p in "${patterns[@]}"; do
        if [[ -n "$result" ]]; then
            result="$result|$p"
        else
            result="$p"
        fi
    done
    echo "$result"
}

# Analyze a single extracted log directory
analyze_node_logs() {
    local node_name="$1"
    local extract_dir="$2"
    local report_file="$3"

    echo "" >> "$report_file"
    echo "================================================================================" >> "$report_file"
    echo "NODE: $node_name" >> "$report_file"
    echo "================================================================================" >> "$report_file"

    local found_issues=false

    # Build search patterns
    local error_pattern
    error_pattern=$(build_pattern ERROR_PATTERNS)

    local search_pattern="$error_pattern"
    if [[ "$INCLUDE_WARNINGS" == "true" ]]; then
        local warning_pattern
        warning_pattern=$(build_pattern WARNING_PATTERNS)
        search_pattern="$search_pattern|$warning_pattern"
    fi

    # Search for component-specific logs
    for component in "${COMPONENTS[@]}"; do
        log_verbose "[$node_name] Searching for $component issues..."

        local component_errors=""
        local error_count=0

        # Search in pod logs directory
        if [[ -d "$extract_dir" ]]; then
            # Find files related to this component
            local component_files
            component_files=$(find "$extract_dir" -type f \( -name "*${component}*" -o -name "*log*" \) 2>/dev/null || true)

            for file in $component_files; do
                if [[ -f "$file" && -r "$file" ]]; then
                    # Check if file contains this component and has errors
                    local matches
                    matches=$(grep -iE "$component" "$file" 2>/dev/null | grep -iE "$search_pattern" 2>/dev/null | head -n "$MAX_ERRORS_PER_COMPONENT" || true)

                    if [[ -n "$matches" ]]; then
                        local relative_file="${file#$extract_dir/}"
                        component_errors+=$'\n'"--- File: $relative_file ---"$'\n'"$matches"
                        error_count=$((error_count + $(echo "$matches" | wc -l)))
                        found_issues=true
                    fi
                fi
            done

            # Also search all log files for component mentions with errors
            local all_logs
            all_logs=$(find "$extract_dir" -type f \( -name "*.log" -o -name "*.txt" -o -path "*/podlogs/*" \) 2>/dev/null || true)

            for file in $all_logs; do
                if [[ -f "$file" && -r "$file" ]]; then
                    local matches
                    matches=$(grep -iE "$component.*($search_pattern)|($search_pattern).*$component" "$file" 2>/dev/null | head -n "$MAX_ERRORS_PER_COMPONENT" || true)

                    if [[ -n "$matches" ]]; then
                        local relative_file="${file#$extract_dir/}"
                        # Avoid duplicates
                        if [[ "$component_errors" != *"$relative_file"* ]]; then
                            component_errors+=$'\n'"--- File: $relative_file ---"$'\n'"$matches"
                            error_count=$((error_count + $(echo "$matches" | wc -l)))
                            found_issues=true
                        fi
                    fi
                fi
            done
        fi

        if [[ -n "$component_errors" && "$error_count" -gt 0 ]]; then
            echo "" >> "$report_file"
            echo "--------------------------------------------------------------------------------" >> "$report_file"
            echo "COMPONENT: $component ($error_count issues found)" >> "$report_file"
            echo "--------------------------------------------------------------------------------" >> "$report_file"
            echo "$component_errors" >> "$report_file"
        fi
    done

    # Search for general Kubernetes issues
    log_verbose "[$node_name] Searching for general Kubernetes issues..."

    local k8s_issues=""
    local k8s_patterns="CrashLoopBackOff|ImagePullBackOff|OOMKilled|Evicted|FailedScheduling|FailedMount|NodeNotReady|PodInitializing|ContainerCreating"

    if [[ -d "$extract_dir" ]]; then
        local kubectl_files
        kubectl_files=$(find "$extract_dir" -type f -name "*kubectl*" -o -name "*pods*" -o -name "*events*" -o -name "*describe*" 2>/dev/null || true)

        for file in $kubectl_files; do
            if [[ -f "$file" && -r "$file" ]]; then
                local matches
                matches=$(grep -iE "$k8s_patterns" "$file" 2>/dev/null | head -n "$MAX_ERRORS_PER_COMPONENT" || true)

                if [[ -n "$matches" ]]; then
                    local relative_file="${file#$extract_dir/}"
                    k8s_issues+=$'\n'"--- File: $relative_file ---"$'\n'"$matches"
                    found_issues=true
                fi
            fi
        done
    fi

    if [[ -n "$k8s_issues" ]]; then
        echo "" >> "$report_file"
        echo "--------------------------------------------------------------------------------" >> "$report_file"
        echo "KUBERNETES ISSUES (pods, events, scheduling)" >> "$report_file"
        echo "--------------------------------------------------------------------------------" >> "$report_file"
        echo "$k8s_issues" >> "$report_file"
    fi

    # Search for etcd issues
    log_verbose "[$node_name] Searching for etcd issues..."

    local etcd_issues=""
    local etcd_patterns="etcd.*error|etcd.*failed|etcd.*timeout|compaction|defrag|leader.*lost|raft|snapshot.*failed"

    if [[ -d "$extract_dir" ]]; then
        local etcd_files
        etcd_files=$(find "$extract_dir" -type f \( -name "*etcd*" -o -path "*/etcd/*" \) 2>/dev/null || true)

        for file in $etcd_files; do
            if [[ -f "$file" && -r "$file" ]]; then
                local matches
                matches=$(grep -iE "$etcd_patterns" "$file" 2>/dev/null | grep -iE "$error_pattern" | head -n "$MAX_ERRORS_PER_COMPONENT" || true)

                if [[ -n "$matches" ]]; then
                    local relative_file="${file#$extract_dir/}"
                    etcd_issues+=$'\n'"--- File: $relative_file ---"$'\n'"$matches"
                    found_issues=true
                fi
            fi
        done
    fi

    if [[ -n "$etcd_issues" ]]; then
        echo "" >> "$report_file"
        echo "--------------------------------------------------------------------------------" >> "$report_file"
        echo "ETCD ISSUES" >> "$report_file"
        echo "--------------------------------------------------------------------------------" >> "$report_file"
        echo "$etcd_issues" >> "$report_file"
    fi

    if [[ "$found_issues" == "false" ]]; then
        echo "" >> "$report_file"
        echo "No significant issues found for monitored components." >> "$report_file"
    fi
}

# Generate summary statistics
generate_summary() {
    local report_file="$1"
    local tmp_report="$2"

    echo "================================================================================" >> "$report_file"
    echo "                        LOG ANALYSIS SUMMARY" >> "$report_file"
    echo "================================================================================" >> "$report_file"
    echo "" >> "$report_file"
    echo "Analysis Date: $(date)" >> "$report_file"
    echo "Logs Directory: $LOGS_DIR" >> "$report_file"
    echo "Include Warnings: $INCLUDE_WARNINGS" >> "$report_file"
    echo "" >> "$report_file"

    # Count issues per component
    echo "ISSUE COUNTS BY COMPONENT:" >> "$report_file"
    echo "--------------------------" >> "$report_file"

    for component in "${COMPONENTS[@]}"; do
        local count
        count=$(grep -c "COMPONENT: $component" "$tmp_report" 2>/dev/null | head -1 || echo "0")
        count=${count:-0}
        if [[ "$count" =~ ^[0-9]+$ ]] && [[ "$count" -gt 0 ]]; then
            local issue_count
            issue_count=$(grep "COMPONENT: $component" "$tmp_report" | grep -oE '\([0-9]+ issues' | grep -oE '[0-9]+' | awk '{sum+=$1} END {print sum}' || echo "0")
            printf "  %-30s %s issues across %s nodes\n" "$component:" "${issue_count:-0}" "$count" >> "$report_file"
        fi
    done

    # Count K8s issues
    local k8s_count
    k8s_count=$(grep -c "KUBERNETES ISSUES" "$tmp_report" 2>/dev/null | head -1 || echo "0")
    k8s_count=${k8s_count:-0}
    if [[ "$k8s_count" =~ ^[0-9]+$ ]] && [[ "$k8s_count" -gt 0 ]]; then
        printf "  %-30s found on %s nodes\n" "Kubernetes Issues:" "$k8s_count" >> "$report_file"
    fi

    # Count etcd issues
    local etcd_count
    etcd_count=$(grep -c "ETCD ISSUES" "$tmp_report" 2>/dev/null | head -1 || echo "0")
    etcd_count=${etcd_count:-0}
    if [[ "$etcd_count" =~ ^[0-9]+$ ]] && [[ "$etcd_count" -gt 0 ]]; then
        printf "  %-30s found on %s nodes\n" "etcd Issues:" "$etcd_count" >> "$report_file"
    fi

    echo "" >> "$report_file"

    # Most common errors
    echo "MOST COMMON ERROR PATTERNS:" >> "$report_file"
    echo "---------------------------" >> "$report_file"

    # Extract and count unique error messages
    grep -hiE "error|Error|ERROR|failed|Failed|FAILED" "$tmp_report" 2>/dev/null | \
        sed 's/^[[:space:]]*//' | \
        sort | uniq -c | sort -rn | head -20 >> "$report_file" || echo "  (none found)" >> "$report_file"

    echo "" >> "$report_file"
}

main() {
    parse_args "$@"

    # Validate logs directory
    if [[ ! -d "$LOGS_DIR" ]]; then
        echo "Error: Logs directory does not exist: $LOGS_DIR" >&2
        exit 1
    fi

    # Find all tarballs
    mapfile -t tarballs < <(find "$LOGS_DIR" -name "*.tar.gz" -type f 2>/dev/null | sort)

    if [[ ${#tarballs[@]} -eq 0 ]]; then
        echo "Error: No log tarballs found in $LOGS_DIR" >&2
        exit 1
    fi

    log "Found ${#tarballs[@]} log archives to analyze"

    # Create temporary working directory
    local work_dir
    work_dir=$(mktemp -d)
    trap "rm -rf '$work_dir'" EXIT

    # Create temporary report file
    local tmp_report="$work_dir/report.txt"
    touch "$tmp_report"

    # Process each tarball
    for tarball in "${tarballs[@]}"; do
        local node_name
        node_name=$(basename "$tarball" | sed 's/-[^-]*-[0-9_-]*\.tar\.gz$//')

        log "Analyzing: $node_name"

        # Create extraction directory
        local extract_dir="$work_dir/$node_name"
        mkdir -p "$extract_dir"

        # Extract tarball
        if tar -xzf "$tarball" -C "$extract_dir" 2>/dev/null; then
            analyze_node_logs "$node_name" "$extract_dir" "$tmp_report"
        else
            echo "" >> "$tmp_report"
            echo "ERROR: Failed to extract $tarball" >> "$tmp_report"
        fi

        # Clean up extracted files to save space
        rm -rf "$extract_dir"
    done

    # Generate final report
    local final_report="$work_dir/final_report.txt"

    # Add summary at the top
    generate_summary "$final_report" "$tmp_report"

    # Append detailed findings
    echo "" >> "$final_report"
    echo "================================================================================" >> "$final_report"
    echo "                        DETAILED FINDINGS BY NODE" >> "$final_report"
    echo "================================================================================" >> "$final_report"
    cat "$tmp_report" >> "$final_report"

    # Output report
    if [[ -n "$OUTPUT_FILE" ]]; then
        cp "$final_report" "$OUTPUT_FILE"
        log "Report written to: $OUTPUT_FILE"
    else
        cat "$final_report"
    fi

    log "Analysis complete"
}

main "$@"
