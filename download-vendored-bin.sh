#!/usr/bin/env bash

set -e

# renovate: datasource=github-release-attachments depName=opentofu/opentofu
OPENTOFU_VERSION=1.11.5
# renovate-local: kubectl
KUBECTL_VERSION=1.35.3
# renovate-local: helm
HELM_VERSION=4.1.3
# renovate: datasource=github-release-attachments depName=k3d-io/k3d
K3D_VERSION=5.8.3

GOOS=`go env GOOS`
GOARCH=`go env GOARCH`

verify_sha256() {
	local archive="$1"
	local checksums_file="$2"
	local result

	if command -v sha256sum >/dev/null 2>&1; then
		# Extract hash and reconstruct checksum line with just the filename (strip any path prefixes)
		awk -v archive="${archive}" '$0 ~ archive "$" { print $1 "  " archive }' "${checksums_file}" | sha256sum -c >/dev/null 2>&1
		result=$?
	elif command -v shasum >/dev/null 2>&1; then
		local expected actual
		expected=$(awk -v archive="${archive}" '$0 ~ archive "$" { print $1 }' "${checksums_file}")
		actual=$(shasum -a 256 "${archive}" | awk '{print $1}')
		[[ "${actual}" == "${expected}" ]]
		result=$?
	else
		echo "No SHA256 tool found (expected sha256sum or shasum)" >&2
		exit 1
	fi

	if [[ ${result} -eq 0 ]]; then
		echo "SHA256 verification succeeded for ${archive}"
	else
		echo "SHA256 verification FAILED for ${archive}" >&2
		exit 1
	fi
}

verify_sha256_digest() {
	local archive="$1"
	local checksum_file="$2"
	local expected actual

	expected=$(tr -d '\r\n' < "${checksum_file}")

	if command -v sha256sum >/dev/null 2>&1; then
		actual=$(sha256sum "${archive}" | awk '{print $1}')
	elif command -v shasum >/dev/null 2>&1; then
		actual=$(shasum -a 256 "${archive}" | awk '{print $1}')
	else
		echo "No SHA256 tool found (expected sha256sum or shasum)" >&2
		exit 1
	fi

	if [[ "${actual}" == "${expected}" ]]; then
		echo "SHA256 verification succeeded for ${archive}"
	else
		echo "SHA256 verification FAILED for ${archive}" >&2
		exit 1
	fi
}

rm -rf internal/vendored/bin
mkdir -p internal/vendored/bin
cd internal/vendored/bin
export PATH="$(pwd):${PATH}"


echo Downloading and unpacking OpenTofu...
OPENTOFU_FILENAME="tofu_${OPENTOFU_VERSION}_${GOOS}_${GOARCH}"
OPENTOFU_ARCHIVE="${OPENTOFU_FILENAME}.zip"
OPENTOFU_URL="https://github.com/opentofu/opentofu/releases/download/v${OPENTOFU_VERSION}/${OPENTOFU_ARCHIVE}"
OPENTOFU_SHA256SUMS="tofu_${OPENTOFU_VERSION}_SHA256SUMS"
OPENTOFU_SHA256SUMS_URL="https://github.com/opentofu/opentofu/releases/download/v${OPENTOFU_VERSION}/${OPENTOFU_SHA256SUMS}"
curl --output ${OPENTOFU_ARCHIVE} --location --fail ${OPENTOFU_URL}
curl --output ${OPENTOFU_SHA256SUMS} --location --fail ${OPENTOFU_SHA256SUMS_URL}
verify_sha256 "${OPENTOFU_ARCHIVE}" "${OPENTOFU_SHA256SUMS}"
rm -f "${OPENTOFU_SHA256SUMS}"
mkdir ${OPENTOFU_FILENAME}
unzip ${OPENTOFU_ARCHIVE} -d ${OPENTOFU_FILENAME}
mv ${OPENTOFU_FILENAME}/tofu .
rm -rf ${OPENTOFU_FILENAME}*

echo Downloading kubectl...
if [[ "${GOARCH}" != "amd64" && "${GOARCH}" != "arm64" ]]; then
	echo "Unsupported kubectl architecture for checksum validation: ${GOARCH}" >&2
	exit 1
fi

KUBECTL_BINARY="kubectl"
if [[ "${GOOS}" == "windows" ]]; then
	KUBECTL_BINARY="kubectl.exe"
fi

KUBECTL_URL="https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/${GOOS}/${GOARCH}/${KUBECTL_BINARY}"
KUBECTL_SHA256_URL="${KUBECTL_URL}.sha256"
KUBECTL_SHA256_FILE="${KUBECTL_BINARY}.sha256"
curl --output ${KUBECTL_BINARY} --location --fail ${KUBECTL_URL}
curl --output ${KUBECTL_SHA256_FILE} --location --fail ${KUBECTL_SHA256_URL}
verify_sha256_digest "${KUBECTL_BINARY}" "${KUBECTL_SHA256_FILE}"
rm -f "${KUBECTL_SHA256_FILE}"

echo Downloading and unpacking Helm...
HELM_FILENAME="helm-v${HELM_VERSION}-${GOOS}-${GOARCH}"
HELM_ARCHIVE="${HELM_FILENAME}.tar.gz"
HELM_URL="https://get.helm.sh/${HELM_ARCHIVE}"
HELM_SHA256SUM_FILE="${HELM_ARCHIVE}.sha256sum"
HELM_SHA256SUM_URL="https://get.helm.sh/${HELM_SHA256SUM_FILE}"
curl --output ${HELM_ARCHIVE} --location --fail ${HELM_URL}
curl --output ${HELM_SHA256SUM_FILE} --location --fail ${HELM_SHA256SUM_URL}
verify_sha256 "${HELM_ARCHIVE}" "${HELM_SHA256SUM_FILE}"
rm -f "${HELM_SHA256SUM_FILE}"
tar xvf ${HELM_ARCHIVE}
mv ${GOOS}-${GOARCH}/helm .
rm -rf ${HELM_ARCHIVE}*
rm -rf ${GOOS}-${GOARCH}*

echo Downloading k3d...
K3D_BINARY="k3d-${GOOS}-${GOARCH}"
K3D_URL="https://github.com/k3d-io/k3d/releases/download/v${K3D_VERSION}/${K3D_BINARY}"
K3D_CHECKSUMS="k3d-v${K3D_VERSION}-checksums.txt"
K3D_CHECKSUMS_URL="https://github.com/k3d-io/k3d/releases/download/v${K3D_VERSION}/checksums.txt"
curl --output ${K3D_BINARY} --location --fail ${K3D_URL}
curl --output ${K3D_CHECKSUMS} --location --fail ${K3D_CHECKSUMS_URL}
verify_sha256 "${K3D_BINARY}" "${K3D_CHECKSUMS}"
rm -f "${K3D_CHECKSUMS}"
mv ${K3D_BINARY} k3d
