#!/usr/bin/env bash
# download-deployment-artifacts.sh
#
# Downloads kubeconfig(s) and rendered-dart.yaml for a dartboard deployment from S3.
#
# Usage:
#   ./download-deployment-artifacts.sh [OPTIONS] <folder-name>
#
# Arguments:
#   <folder-name>     The deployment folder name in S3 (i.e. the DEPLOYMENT_ID / project name).
#
# Options:
#   -p, --profile     AWS CLI profile to use (overrides standard AWS_PROFILE env var)
#   -b, --bucket      S3 bucket name (or set S3_BUCKET_NAME env var)
#   -r, --region      AWS region of the S3 bucket (or set S3_BUCKET_REGION env var)
#   -o, --output-dir  Local directory to write files into (default: ./<folder-name>)
#   -h, --help        Show this help text
#
# Authentication:
#   Standard AWS credential chain is used (env vars, ~/.aws/credentials, instance profile, etc.).
#   Required: AWS_ACCESS_KEY_ID + AWS_SECRET_ACCESS_KEY, or a configured AWS profile.
#
# Requirements:
#   aws CLI v2  (https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
#   unzip

set -euo pipefail

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
usage() {
  sed -n '/^# Usage:/,/^[^#]/p' "$0" | sed 's/^# \{0,2\}//' | head -n -1
  exit "${1:-0}"
}

die() { echo "ERROR: $*" >&2; exit 1; }
info() { echo "==> $*"; }

# ---------------------------------------------------------------------------
# Parse arguments
# ---------------------------------------------------------------------------
AWS_PROFILE="${AWS_PROFILE:-}"
BUCKET="${S3_BUCKET_NAME:-}"
REGION="${S3_BUCKET_REGION:-}"
OUTPUT_DIR=""
FOLDER=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -p|--profile)  AWS_PROFILE="$2"; shift 2 ;;
    -b|--bucket)   BUCKET="$2";     shift 2 ;;
    -r|--region)   REGION="$2";     shift 2 ;;
    -o|--output-dir) OUTPUT_DIR="$2"; shift 2 ;;
    -h|--help)     usage 0 ;;
    -*)            die "Unknown option: $1" ;;
    *)
      [[ -z "$FOLDER" ]] || die "Unexpected extra argument: $1"
      FOLDER="$1"
      shift
      ;;
  esac
done

# ---------------------------------------------------------------------------
# Validate inputs
# ---------------------------------------------------------------------------
[[ -n "$AWS_PROFILE" ]] || die "AWS profile not specified. Pass -p/--profile or set AWS_PROFILE environment variable."
[[ -n "$FOLDER" ]]  || die "Missing required argument: <folder-name>. Run with --help for usage."
[[ -n "$BUCKET" ]]  || die "S3 bucket name is required. Pass -b/--bucket or set S3_BUCKET_NAME."
[[ -n "$REGION" ]]  || die "S3 region is required. Pass -r/--region or set S3_BUCKET_REGION."

command -v aws   >/dev/null 2>&1 || die "aws CLI not found. Install it from https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html"
command -v unzip >/dev/null 2>&1 || die "unzip not found. Install it with your package manager."

S3_PREFIX="s3://${BUCKET}/${FOLDER}"

if [[ -z "$OUTPUT_DIR" ]]; then
  OUTPUT_DIR="./${FOLDER}"
fi

mkdir -p "$OUTPUT_DIR"
info "Writing artifacts to: $(realpath "$OUTPUT_DIR")"

# ---------------------------------------------------------------------------
# Helper: download one S3 object, exit cleanly if it doesn't exist
# ---------------------------------------------------------------------------
s3_download() {
  local src="$1" dst="$2"
  if aws --profile "$AWS_PROFILE" s3 cp "$src" "$dst" --region "$REGION" 2>/dev/null; then
    info "Downloaded: $(basename "$dst")"
  else
    echo "  WARN: $src not found — skipping." >&2
  fi
}

# ---------------------------------------------------------------------------
# 1. rendered-dart.yaml
# ---------------------------------------------------------------------------
info "Fetching rendered-dart.yaml ..."
s3_download "${S3_PREFIX}/rendered-dart.yaml" "${OUTPUT_DIR}/rendered-dart.yaml"

# ---------------------------------------------------------------------------
# 2. access-details.log  (optional, useful for context)
# ---------------------------------------------------------------------------
info "Fetching access-details.log ..."
s3_download "${S3_PREFIX}/access-details.log" "${OUTPUT_DIR}/access-details.log"

# ---------------------------------------------------------------------------
# 3. Config zip  → extract kubeconfig(s)
#
#    The zip is named  <project>_config.zip  and lives at the root of the
#    S3 folder.  We discover its exact name by listing the prefix.
# ---------------------------------------------------------------------------
info "Looking for *_config.zip in ${S3_PREFIX}/ ..."

CONFIG_ZIP_KEY=$(
  aws --profile "$AWS_PROFILE" s3 ls "${S3_PREFIX}/" --region "$REGION" 2>/dev/null \
    | awk '{print $NF}' \
    | grep '_config\.zip$' \
    | head -n 1 \
    || true
)

if [[ -z "$CONFIG_ZIP_KEY" ]]; then
  echo "  WARN: No *_config.zip found under ${S3_PREFIX}/ — kubeconfig(s) not downloaded." >&2
else
  ZIPFILE="${OUTPUT_DIR}/${CONFIG_ZIP_KEY}"
  info "Downloading ${CONFIG_ZIP_KEY} ..."
  aws --profile "$AWS_PROFILE" s3 cp "${S3_PREFIX}/${CONFIG_ZIP_KEY}" "$ZIPFILE" --region "$REGION"

  # Extract only *.yaml files (the kubeconfigs); skip SSH scripts, etc.
  KUBECONFIG_DIR="${OUTPUT_DIR}/kubeconfigs"
  mkdir -p "$KUBECONFIG_DIR"
  info "Extracting kubeconfig(s) from ${CONFIG_ZIP_KEY} to ${KUBECONFIG_DIR}/ ..."
  unzip -o -j "$ZIPFILE" "*.yaml" -d "$KUBECONFIG_DIR"

  KUBE_COUNT=$(find "$KUBECONFIG_DIR" -name "*.yaml" | wc -l | tr -d ' ')
  if [[ "$KUBE_COUNT" -eq 0 ]]; then
    echo "  WARN: No *.yaml files found inside the config zip." >&2
  else
    info "Extracted ${KUBE_COUNT} kubeconfig file(s):"
    find "$KUBECONFIG_DIR" -name "*.yaml" | sort | sed 's/^/    /'
  fi

  rm -f "$ZIPFILE"
fi

# ---------------------------------------------------------------------------
# 4. Tfstate zip  → extract tfstate files
#
#    The zip is named  <project>_tfstate.zip  and lives at the root of the
#    S3 folder.  We discover its exact name by listing the prefix.
# ---------------------------------------------------------------------------
info "Looking for *tfstate*.zip in ${S3_PREFIX}/ ..."

TFSTATE_ZIP_KEY=$(
  aws --profile "$AWS_PROFILE" s3 ls "${S3_PREFIX}/" --region "$REGION" 2>/dev/null \
    | awk '{print $NF}' \
    | grep 'tfstate.*\.zip$' \
    | head -n 1 \
    || true
)

if [[ -z "$TFSTATE_ZIP_KEY" ]]; then
  echo "  WARN: No *tfstate*.zip found under ${S3_PREFIX}/ — tfstate file(s) not downloaded." >&2
else
  TFSTATE_ZIPFILE="${OUTPUT_DIR}/${TFSTATE_ZIP_KEY}"
  info "Downloading ${TFSTATE_ZIP_KEY} ..."
  aws --profile "$AWS_PROFILE" s3 cp "${S3_PREFIX}/${TFSTATE_ZIP_KEY}" "$TFSTATE_ZIPFILE" --region "$REGION"

  # The zip preserves the workspace directory structure: <workspace>/terraform.tfstate
  # Extract without -j so that structure is intact.  To use with dartboard destroy, the
  # contents must land at:  <tofu_main_directory>/terraform.tfstate.d/<workspace>/
  # Point -d at that parent directory if you know it, or extract here and move manually.
  TFSTATE_DIR="${OUTPUT_DIR}/tfstate"
  mkdir -p "$TFSTATE_DIR"
  info "Extracting tfstate file(s) from ${TFSTATE_ZIP_KEY} to ${TFSTATE_DIR}/ ..."
  unzip -o "$TFSTATE_ZIPFILE" -d "$TFSTATE_DIR"

  TFSTATE_COUNT=$(find "$TFSTATE_DIR" -type f | wc -l | tr -d ' ')
  if [[ "$TFSTATE_COUNT" -eq 0 ]]; then
    echo "  WARN: No files found inside the tfstate zip." >&2
  else
    info "Extracted ${TFSTATE_COUNT} tfstate file(s):"
    find "$TFSTATE_DIR" -type f | sort | sed 's/^/    /'
    echo "" >&2
    echo "  NOTE: To run 'dartboard destroy', copy the extracted workspace directory into" >&2
    echo "  the dartboard repo at: <tofu_main_directory>/terraform.tfstate.d/" >&2
    echo "  e.g.: cp -r ${TFSTATE_DIR}/*/ dartboard/tofu/main/aws/terraform.tfstate.d/" >&2
  fi

  rm -f "$TFSTATE_ZIPFILE"
fi

# ---------------------------------------------------------------------------
# If nothing was downloaded, list available folders to help identify the right name
# ---------------------------------------------------------------------------
if [[ ! -f "${OUTPUT_DIR}/rendered-dart.yaml" && ! -f "${OUTPUT_DIR}/access-details.log" && -z "$CONFIG_ZIP_KEY" && -z "$TFSTATE_ZIP_KEY" ]]; then
  echo "" >&2
  echo "  No artifacts were downloaded. Available top-level folders in s3://${BUCKET}/:" >&2
  aws --profile "$AWS_PROFILE" s3 ls "s3://${BUCKET}/" --region "$REGION" 2>/dev/null \
    | awk '/\/$/ {print "    " $NF}' \
    || echo "  (could not list bucket contents — check your credentials and bucket name)" >&2
fi

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------
echo ""
echo "Done. Artifacts written to: $(realpath "$OUTPUT_DIR")"
echo ""
echo "To use a kubeconfig:"
echo "  export KUBECONFIG=\"$(realpath "$OUTPUT_DIR")/kubeconfigs/<cluster>.yaml\""

