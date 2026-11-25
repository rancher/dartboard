#!/usr/bin/env bash

# Script to determine compatible component versions for a given Rancher version.
# Dependencies: curl, jq

set -e

if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed."
    exit 1
fi

POSITIONAL_ARGS=()
DEV_MODE=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --dev)
      DEV_MODE=true
      shift # past argument
      ;;
    *)
      POSITIONAL_ARGS+=("$1")
      shift # past argument
      ;;
  esac
done

set -- "${POSITIONAL_ARGS[@]}" # restore positional parameters

if [ -z "$1" ]; then
    echo "Usage: $0 <rancher-version> [--dev]"
    echo "Example: $0 v2.14.0 --dev"
    exit 1
fi

RANCHER_VERSION=$1
# Remove 'v' prefix if present for branch calculation
VERSION_NUM=${RANCHER_VERSION#v}
MAJOR_MINOR=$(echo "$VERSION_NUM" | cut -d. -f1-2)
# Rancher git tags always carry the 'v' prefix
RANCHER_VERSION="v${VERSION_NUM}"

if [[ "$RANCHER_VERSION" =~ (rc|alpha|beta|dev) ]]; then
    DEV_MODE=true
fi

if [ "$DEV_MODE" = true ]; then
    PRIMARY_BRANCH="dev-v${MAJOR_MINOR}"
    FALLBACK_BRANCH="release-v${MAJOR_MINOR}"
else
    PRIMARY_BRANCH="release-v${MAJOR_MINOR}"
    FALLBACK_BRANCH="dev-v${MAJOR_MINOR}"
fi

echo "=================================================="
echo "Rancher Version: $RANCHER_VERSION"
echo "Dev Mode: $DEV_MODE"
echo "=================================================="

# --- Rancher Helm Repository ---
# Mirrors dartboard's chartInstallRancher: "alpha" in the version -> rancher-alpha, else rancher-latest.
echo ""
echo "--- Rancher Helm Repository ---"
CHART_BASE="https://releases.rancher.com/server-charts"
if [[ "$VERSION_NUM" == *alpha* ]]; then DEFAULT_REPO="alpha"; else DEFAULT_REPO="latest"; fi
echo "Dartboard default: rancher-$DEFAULT_REPO $CHART_BASE/$DEFAULT_REPO"
echo "(dartboard uses prime https://charts.rancher.com/server-charts/prime when force_prime_registry is true;"
echo " chart_variables.rancher_chart_repo_override takes precedence over everything)"

# shortcut: HEAD on the tgz only; stable vs latest overlap, cannot be told apart from the version string.
FOUND_REPOS=""
for repo in stable latest alpha; do
    if curl -sfI --max-time 10 -o /dev/null "$CHART_BASE/$repo/rancher-${VERSION_NUM}.tgz"; then
        FOUND_REPOS="$FOUND_REPOS $repo"
    fi
done
FOUND_REPOS=${FOUND_REPOS# }
if [ -z "$FOUND_REPOS" ]; then
    echo "Not verified: rancher-${VERSION_NUM}.tgz not found in stable/latest/alpha (or network check failed)."
elif [[ " $FOUND_REPOS " == *" $DEFAULT_REPO "* ]]; then
    echo "Verified: rancher-${VERSION_NUM}.tgz is available in: $FOUND_REPOS (includes default rancher-$DEFAULT_REPO)"
else
    FIRST_REPO=${FOUND_REPOS%% *}
    echo "WARNING: rancher-${VERSION_NUM}.tgz is NOT in the dartboard default rancher-$DEFAULT_REPO."
    echo "Available in: $FOUND_REPOS. Set chart_variables.rancher_chart_repo_override to $CHART_BASE/$FIRST_REPO/rancher-"
fi

# For -rc versions, also report the newest "<base>-<commit>-head" dev build of the same release (optimus staging repo).
if [[ "$VERSION_NUM" == *-rc* ]]; then
    HEAD_BASE="https://charts.optimus.rancher.io/server-charts/latest"
    # created precedes version within each index entry; ISO timestamps sort lexically.
    LATEST_HEAD=$(curl -sf --max-time 30 "$HEAD_BASE/index.yaml" \
        | awk -v base="${VERSION_NUM%%-rc*}" '/^ +created:/ {c=$2} $1=="version:" && $2 ~ "^" base "-[0-9a-f]+-head$" {print c, $2}' \
        | sort | tail -n 1 | cut -d' ' -f2 || true)
    if [ -n "$LATEST_HEAD" ]; then
        echo "Latest head build for ${VERSION_NUM%%-rc*}: $LATEST_HEAD"
        echo "Set chart_variables.rancher_chart_repo_override to $HEAD_BASE/rancher- and rancher_version to $LATEST_HEAD"
    else
        echo "No ${VERSION_NUM%%-rc*}-<commit>-head chart found in $HEAD_BASE (or network check failed)."
    fi
fi
echo ""

# --- Kubernetes Versions (K3s & RKE2) ---
echo "Fetching Kubernetes versions from Kontainer Driver Metadata (KDM)..."

# Try primary branch first, fall back to fallback branch if not found
KDM_URL="https://raw.githubusercontent.com/rancher/kontainer-driver-metadata/${PRIMARY_BRANCH}/data/data.json"
HTTP_STATUS=$(curl -o /dev/null -s -w "%{http_code}\n" "$KDM_URL")
RELEASE_BRANCH="$PRIMARY_BRANCH"

if [ "$HTTP_STATUS" != "200" ]; then
    echo "Primary branch ($PRIMARY_BRANCH) KDM not found, trying fallback ($FALLBACK_BRANCH)..."
    KDM_URL="https://raw.githubusercontent.com/rancher/kontainer-driver-metadata/${FALLBACK_BRANCH}/data/data.json"
    HTTP_STATUS=$(curl -o /dev/null -s -w "%{http_code}\n" "$KDM_URL")
    if [ "$HTTP_STATUS" == "200" ]; then
        RELEASE_BRANCH="$FALLBACK_BRANCH"
    fi
fi

echo "Targeting Branch: $RELEASE_BRANCH"
KDM_DATA=$(curl -s "$KDM_URL")

if [ -z "$KDM_DATA" ]; then
    echo "Error: Failed to fetch KDM data."
else
    # Fetch Rancher Chart.yaml to check for kubeVersion constraints
    RANCHER_CHART_URL="https://raw.githubusercontent.com/rancher/rancher/${RANCHER_VERSION}/chart/Chart.yaml"
    CHART_YAML=$(curl -sL "$RANCHER_CHART_URL")
    MAX_K8S_VERSION=""

    if [[ "$CHART_YAML" == *"kubeVersion:"* ]]; then
        KUBE_CONSTRAINT=$(echo "$CHART_YAML" | grep "^kubeVersion:" | sed 's/kubeVersion://g' | tr -d '"' | tr -d "'")
        KUBE_CONSTRAINT=$(echo "$KUBE_CONSTRAINT" | sed 's/&lt;/</g; s/&gt;/>/g')
        echo "Found kubeVersion constraint in Rancher Chart: $KUBE_CONSTRAINT"
        
        if [[ "$KUBE_CONSTRAINT" == *"<"* ]]; then
            # Extract the version after the last '<' (handles '< 1.35.0' and '>= 1.28.0 < 1.35.0')
            MAX_K8S_VERSION=$(echo "$KUBE_CONSTRAINT" | awk -F'<' '{print $NF}' | awk '{print $1}')
            echo "Applying max version filter: < $MAX_K8S_VERSION"
        fi
    fi

    filter_versions() {
        local versions="$1"
        local max_ver="$2"
        
        if [ -z "$max_ver" ]; then
            echo "$versions"
            return
        fi
        
        for v in $versions; do
            local v_clean=${v#v}
            local v_base=${v_clean%+*}
            local max_clean=${max_ver#v}

            if [[ "$max_clean" == *"-0" ]]; then
                local max_base=${max_clean%-0}
                if [ "$v_base" = "$max_base" ]; then continue; fi
            fi
            
            if [ "$v_base" = "$max_clean" ]; then
                continue
            fi
            
            local sorted=$(printf "%s\n%s" "$v_base" "$max_clean" | sort -V | head -n 1)
            if [ "$sorted" = "$v_base" ]; then
                echo "$v"
            fi
        done
    }

    echo ""
    echo "--- K3s Versions (Top 5 Stable) ---"
    if [ "$DEV_MODE" = true ]; then
        VERSIONS=$(echo "$KDM_DATA" | jq -r '.k3s.releases[].version' | sort -V -r)
    else
        VERSIONS=$(echo "$KDM_DATA" | jq -r '.k3s.releases[].version' | grep -v -E 'rc|alpha|beta' | sort -V -r)
    fi
    if [ -n "$MAX_K8S_VERSION" ]; then VERSIONS=$(filter_versions "$VERSIONS" "$MAX_K8S_VERSION"); fi
    echo "$VERSIONS" | head -n 5

    echo ""
    echo "--- RKE2 Versions (Top 5 Stable) ---"
    if [ "$DEV_MODE" = true ]; then
        VERSIONS=$(echo "$KDM_DATA" | jq -r '.rke2.releases[].version' | sort -V -r)
    else
        VERSIONS=$(echo "$KDM_DATA" | jq -r '.rke2.releases[].version' | grep -v -E 'rc|alpha|beta' | sort -V -r)
    fi
    if [ -n "$MAX_K8S_VERSION" ]; then VERSIONS=$(filter_versions "$VERSIONS" "$MAX_K8S_VERSION"); fi
    echo "$VERSIONS" | head -n 5
fi

# Print the body of a URL only on HTTP 200 (quiet, empty output otherwise).
# Avoids GNU-only 'head -n -1'.
fetch_ok_body() {
    local response
    response=$(curl -s -w "\n%{http_code}" "$1") || return 0
    [ "$(echo "$response" | tail -n 1)" = "200" ] || return 0
    echo "$response" | sed '$d'
}

# --- Helper function for Charts ---
get_chart_versions() {
    local chart_name=$1
    local limit=$2
    
    # Use GitHub API to list directories in the charts folder
    # This avoids downloading the huge index.yaml
    local api_url="https://api.github.com/repos/rancher/charts/contents/charts/${chart_name}?ref=${RELEASE_BRANCH}"
    
    local body
    body=$(fetch_ok_body "$api_url")
    [ -n "$body" ] || return 0

    local versions
    versions=$(echo "$body" | jq -r '.[] | select(.type=="dir") | .name')

    if [ "$DEV_MODE" = true ]; then
        echo "$versions" | sort -V -r | head -n "$limit"
    else
        echo "$versions" | grep -v -E 'rc|alpha|beta' | sort -V -r | head -n "$limit"
    fi
}

echo ""
echo "=================================================="
echo "Fetching Chart Versions from rancher/charts ($RELEASE_BRANCH)..."
echo "=================================================="

# --- Rancher Monitoring ---
# Rancher >= 2.15 replaces rancher-monitoring with rancher-monitoring-dashboards on top of an
# upstream kube-prometheus-stack (not shipped in rancher/charts, no version constraint published).
USE_DASHBOARDS=false
if [ "$(printf "%s\n2.15" "$MAJOR_MINOR" | sort -V | head -n 1)" = "2.15" ]; then
    USE_DASHBOARDS=true
fi

if [ "$USE_DASHBOARDS" = true ]; then
    echo ""
    echo "--- Rancher Monitoring Dashboards Versions (Top 5 Stable) ---"
    MONITORING_VERSIONS=$(get_chart_versions "rancher-monitoring-dashboards" 5)
    if [ -z "$MONITORING_VERSIONS" ]; then
        echo "No versions found for rancher-monitoring-dashboards."
    else
        echo "$MONITORING_VERSIONS"
    fi

    echo ""
    echo "--- kube-prometheus-stack Versions (OCI ghcr.io/prometheus-community/charts, as used by dartboard, Top 5 Stable) ---"
    # Same registry dartboard installs from; tags/list is capped at 1000 per page, so page with last=.
    KPS_REPO="prometheus-community/charts/kube-prometheus-stack"
    KPS_TOKEN=$(curl -s "https://ghcr.io/token?scope=repository:${KPS_REPO}:pull" | jq -r '.token // empty')
    KPS_ALL=""
    KPS_LAST=""
    while [ -n "$KPS_TOKEN" ]; do
        KPS_PAGE=$(curl -s -H "Authorization: Bearer $KPS_TOKEN" \
            "https://ghcr.io/v2/${KPS_REPO}/tags/list?n=1000${KPS_LAST:+&last=$KPS_LAST}" | jq -r '.tags[]?' 2>/dev/null || true)
        [ -z "$KPS_PAGE" ] && break
        KPS_ALL+="$KPS_PAGE"$'\n'
        KPS_LAST=$(echo "$KPS_PAGE" | tail -n 1)
    done
    KPS_VERSIONS=$(echo "$KPS_ALL" | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' | sort -V -r | head -n 5 || true)
    if [ -z "$KPS_VERSIONS" ]; then
        echo "No versions found for kube-prometheus-stack."
    else
        echo "$KPS_VERSIONS"
    fi
else
    echo ""
    echo "--- Rancher Monitoring Versions (Top 5 Stable) ---"
    MONITORING_VERSIONS=$(get_chart_versions "rancher-monitoring" 5)

    if [ -z "$MONITORING_VERSIONS" ]; then
        echo "No versions found for rancher-monitoring."
    else
        echo "$MONITORING_VERSIONS"
    fi
fi

# --- Cert Manager ---
echo ""
echo "--- Cert-Manager Versions (Top 5 Stable) ---"

get_upstream_cert_manager_versions() {
    local limit=$1
    local api_url="https://api.github.com/repos/cert-manager/cert-manager/releases"
    
    local body
    body=$(fetch_ok_body "$api_url")
    [ -n "$body" ] || return 0

    local versions
    versions=$(echo "$body" | jq -r '.[].tag_name')

    if [ "$DEV_MODE" = true ]; then
        echo "$versions" | sort -V -r | head -n "$limit"
    else
        echo "$versions" | grep -v -E 'rc|alpha|beta' | sort -V -r | head -n "$limit"
    fi
}

CERT_MANAGER_VERSIONS=$(get_upstream_cert_manager_versions 5)

if [ -z "$CERT_MANAGER_VERSIONS" ]; then
    echo "No versions found for cert-manager."
else
    echo "$CERT_MANAGER_VERSIONS"
fi

if [ "$USE_DASHBOARDS" = true ]; then
    echo ""
    echo "--- Grafana Versions (Based on kube-prometheus-stack Versions) ---"
    if [ -n "$KPS_VERSIONS" ]; then
        printf "%-35s | %s\n" "kube-prometheus-stack Ver" "Grafana Chart Ver"
        printf "%-35s | %s\n" "-----------------------------------" "-----------------"
        for k_ver in $KPS_VERSIONS; do
            KPS_CHART_URL="https://raw.githubusercontent.com/prometheus-community/helm-charts/kube-prometheus-stack-${k_ver}/charts/kube-prometheus-stack/Chart.yaml"
            GRAFANA_DEP_VER=$(curl -s "$KPS_CHART_URL" | grep -A1 -E '^[[:space:]]*- name: grafana$' | grep 'version:' | head -n 1 | awk '{print $2}' | tr -d '"')
            printf "%-35s | %s\n" "$k_ver" "${GRAFANA_DEP_VER:-Not Found}"
        done
    else
        echo "Skipping Grafana check (no kube-prometheus-stack versions found)."
    fi
else
    # --- Grafana (Sub-chart of Monitoring) ---
    echo ""
    echo "--- Grafana Versions (Based on Monitoring Versions) ---"

    if [ -n "$MONITORING_VERSIONS" ]; then
        printf "%-35s | %-20s | %s\n" "Monitoring Ver" "Grafana App Ver" "Grafana Chart Ver"
        printf "%-35s | %-20s | %s\n" "-----------------------------------" "--------------------" "-----------------"
        
        for m_ver in $MONITORING_VERSIONS; do
            # Construct URL to the Grafana subchart Chart.yaml
            # URL encoded + is %2B, but raw github handles + correctly usually.
            
            GRAFANA_CHART_URL="https://raw.githubusercontent.com/rancher/charts/${RELEASE_BRANCH}/charts/rancher-monitoring/${m_ver}/charts/grafana/Chart.yaml"
            
            CHART_YAML=$(curl -s "$GRAFANA_CHART_URL")
            
            if [[ "$CHART_YAML" == *"404: Not Found"* ]]; then
                 printf "%-35s | %-20s | %s\n" "$m_ver" "Not Found" "N/A"
            else
                APP_VER=$(echo "$CHART_YAML" | grep "^appVersion:" | head -n 1 | awk '{print $2}' | tr -d '"')
                CHART_VER=$(echo "$CHART_YAML" | grep "^version:" | head -n 1 | awk '{print $2}' | tr -d '"')
                printf "%-35s | %-20s | %s\n" "$m_ver" "$APP_VER" "$CHART_VER"
            fi
        done
    else
        echo "Skipping Grafana check (no monitoring versions found)."
    fi
fi

echo ""
