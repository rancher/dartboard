#!/usr/bin/env sh
# Requires: git, curl, sha512sum, base64, od, tr, awk, an ssh agent and key with access to the repositories

set -xe
# renovate: datasource=github-tags depName=nodeca/js-yaml
JS_YAML_VERSION=5.4.2
# renovate: datasource=github-tags depName=nodeca/js-yaml digestVersion=5.4.2
JS_YAML_SHA256=a4ed507c9e2840c251364aae5295b4ea23d756c8855eb69e3f819405a2834c70
JS_YAML_COMMIT_HASH=494400bd45cad078123cfc057e674a9a0a8d9983

# renovate: datasource=github-release-attachments depName=grafana/k6-jslib-url
K6_URL_VERSION=1.0.0
# renovate: datasource=github-release-attachments depName=grafana/k6-jslib-url digestVersion=1.0.0
K6_URL_SHA256=43812073b0b2fdb09edd506098ea9247d6d16c373894d8e843864e5df813dddd
K6_URL_COMMIT_HASH=300098d64b41fca72b308bb36ad90df470b74bc0

# renovate-local: k6-jslib-summary
K6_SUMMARY_VERSION=0.1.0
# renovate-local: k6-jslib-summary=0.1.0
K6_SUMMARY_SHA512="PLaveAq5D46LZM24/+vpLcreZegPXzHwemqd/DEpv0wrZpJ64JiyvQ/fj9M6kPqSJkX+5ML2dUJ7iJ7kYvxgZQ=="

# renovate: datasource=github-tags depName=benc-uk/k6-reporter
K6_REPORTER_VERSION=3.0.4
# renovate: datasource=github-tags depName=benc-uk/k6-reporter digestVersion=3.0.4
K6_REPORTER_SHA256=3d981a8f06dc558700e01e3fcbc3ab9214f4d45a4ff43c637c1ddf2e8d22c13c
K6_REPORTER_COMMIT_HASH=25058f7861695cb4fe6e0ecf6415ea489c489005

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

verify_sha256() {
	local file="$1"
	local checksum="$2"
	local expected actual

	expected=$(tr -d '\r\n' <<< "${checksum}")

	if command -v sha256sum >/dev/null 2>&1; then
		actual=$(sha256sum "${file}" | awk '{print $1}')
	elif command -v shasum >/dev/null 2>&1; then
		actual=$(shasum -a 256 "${file}" | awk '{print $1}')
	else
		echo "No SHA256 tool found (expected sha256sum or shasum)" >&2
		exit 1
	fi

	if [[ "${actual}" == "${expected}" ]]; then
		echo "SHA256 verification succeeded for ${file}"
	else
		echo "SHA256 verification FAILED for ${file}" >&2
		exit 1
	fi
}

# Clone js-yaml at specific commit and extract file
tmpdir=$(mktemp -d)
trap "rm -rf $tmpdir" EXIT
git clone --quiet --depth 1 git@github.com:nodeca/js-yaml.git "$tmpdir/js-yaml"
(cd "$tmpdir/js-yaml" && git fetch --quiet origin "$JS_YAML_COMMIT_HASH" && git checkout --quiet "$JS_YAML_COMMIT_HASH")
verify_sha256 "$tmpdir/js-yaml/bin/js-yaml.mjs" "$JS_YAML_SHA256"
cp "$tmpdir/js-yaml/bin/js-yaml.mjs" js-yaml-${JS_YAML_VERSION}.mjs

curl -o k6-summary-${K6_SUMMARY_VERSION}.js https://jslib.k6.io/k6-summary/${K6_SUMMARY_VERSION}/index.js
# Verify checksums for k6-summary
# NOTE: this is just a best-effort tamper detection mechanism, since the file is fetched from a public CDN 
# and not from a private repository. The checksum is stored in the repo, so if the file is tampered with, 
# the checksum in the repo would also need to be updated, which would require a commit and a PR, making it more likely that someone would notice.
if [ -n "$K6_SUMMARY_SHA512" ]; then
    verify_sha512_base64 "$K6_SUMMARY_SHA512" "k6-summary-${K6_SUMMARY_VERSION}.js"
fi

# Clone k6-reporter at specific commit and extract file
git clone --quiet --depth 1 git@github.com:benc-uk/k6-reporter.git "$tmpdir/k6-reporter"
(cd "$tmpdir/k6-reporter" && git fetch --quiet origin "$K6_REPORTER_COMMIT_HASH" && git checkout --quiet "$K6_REPORTER_COMMIT_HASH")
verify_sha256 "$tmpdir/k6-reporter/dist/bundle.js" "$K6_REPORTER_SHA256"
cp "$tmpdir/k6-reporter/dist/bundle.js" k6-reporter-${K6_REPORTER_VERSION}.js

