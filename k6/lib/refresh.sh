#!/usr/bin/env sh
# Requires: git, curl, sha512sum, base64, od, tr, awk, an ssh agent and key with access to the repositories

set -xe
# renovate: datasource=github-tags depName=nodeca/js-yaml
JS_YAML_VERSION=4.1.1
JS_YAML_COMMIT_HASH=50968b862e75866ef90e626572fe0b2f97b55f9f

# renovate-local: k6-summary
K6_SUMMARY_VERSION=0.1.0
# renovate-local: k6-summary=0.1.0
K6_SUMMARY_SHA512="PLaveAq5D46LZM24/+vpLcreZegPXzHwemqd/DEpv0wrZpJ64JiyvQ/fj9M6kPqSJkX+5ML2dUJ7iJ7kYvxgZQ=="

# renovate: datasource=github-tags depName=benc-uk/k6-reporter
K6_REPORTER_VERSION=3.0.4
K6_REPORTER_COMMIT_HASH=25058f7861695cb4fe6e0ecf6415ea489c489005

# Clone js-yaml at specific commit and extract file
tmpdir=$(mktemp -d)
trap "rm -rf $tmpdir" EXIT
git clone --quiet --depth 1 git@github.com:nodeca/js-yaml.git "$tmpdir/js-yaml"
(cd "$tmpdir/js-yaml" && git fetch --quiet origin "$JS_YAML_COMMIT_HASH" && git checkout --quiet "$JS_YAML_COMMIT_HASH")
cp "$tmpdir/js-yaml/dist/js-yaml.mjs" js-yaml-${JS_YAML_VERSION}.mjs
curl -o k6-summary-${K6_SUMMARY_VERSION}.js https://jslib.k6.io/k6-summary/${K6_SUMMARY_VERSION}/index.js

# Clone k6-reporter at specific commit and extract file
git clone --quiet --depth 1 git@github.com:benc-uk/k6-reporter.git "$tmpdir/k6-reporter"
(cd "$tmpdir/k6-reporter" && git fetch --quiet origin "$K6_REPORTER_COMMIT_HASH" && git checkout --quiet "$K6_REPORTER_COMMIT_HASH")
cp "$tmpdir/k6-reporter/dist/bundle.js" k6-reporter-${K6_REPORTER_VERSION}.js

verify_sha512_base64() {
    expected_base64="$1"
    file="$2"

    expected_hex="$(printf '%s' "$expected_base64" | base64 -d 2>/dev/null | od -An -vtx1 | tr -d ' \n')"
    actual_hex="$(sha512sum "$file" | awk '{print $1}')"

    if [ -z "$expected_hex" ] || [ "$expected_hex" != "$actual_hex" ]; then
        echo "checksum mismatch for $file" >&2
        exit 1
    fi
}

# Verify checksums for k6-summary
# NOTE: this is just a best-effort tamper detection mechanism, since the file is fetched from a public CDN 
# and not from a private repository. The checksum is stored in the repo, so if the file is tampered with, 
# the checksum in the repo would also need to be updated, which would require a commit and a PR, making it more likely that someone would notice.
if [ -n "$K6_SUMMARY_SHA512" ]; then
    verify_sha512_base64 "$K6_SUMMARY_SHA512" "k6-summary-${K6_SUMMARY_VERSION}.js"
fi
