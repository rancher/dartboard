#!/usr/bin/env bash

set -e

# renovate: datasource=github-release-attachments depName=opentofu/opentofu
OPENTOFU_VERSION=1.12.6
# renovate-local: kubectl
KUBECTL_VERSION=1.37.0
# renovate-local: helm
HELM_VERSION=4.3.0
# renovate: datasource=github-release-attachments depName=k3d-io/k3d
K3D_VERSION=5.9.0

GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)

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

set_sha_by_arch() {
	local checksums_file=$1
	local arch_key
	case "${GOOS}_${GOARCH}" in
		"linux_amd64")
			arch_key="LINUX_AMD64"
			;;
		"linux_arm64")
			arch_key="LINUX_ARM64"
			;;
		"darwin_amd64")
			arch_key="DARWIN_AMD64"
			;;
		"darwin_arm64")
			arch_key="DARWIN_ARM64"
			;;
		*)
			echo "Unsupported architecture for OpenTofu: ${GOOS}_${GOARCH}" >&2
			exit 1
			;;
	esac
	awk -v key="${arch_key}" '$1 == key { print $2 }' "${checksums_file}"
}

vendor_opentofu() {
	echo "Downloading and unpacking OpenTofu..."
	OPENTOFU_FILENAME="tofu_${OPENTOFU_VERSION}_${GOOS}_${GOARCH}"
	OPENTOFU_ARCHIVE="${OPENTOFU_FILENAME}.tar.gz"
	OPENTOFU_URL="https://github.com/opentofu/opentofu/releases/download/v${OPENTOFU_VERSION}/${OPENTOFU_ARCHIVE}"
	OPENTOFU_SHA256SUMS="${OPENTOFU_FILENAME}_SHA256SUMS"
	# renovate-local: opentofu-linux-amd64
	OPENTOFU_SHA256_LINUX_AMD64="50a6106fa4de523d09c87af85f3db1dd47535fc005727fdca6852146476b88ec"
	# renovate-local: opentofu-linux-arm64
	OPENTOFU_SHA256_LINUX_ARM64="9bd0228a81bcd0c88f7045c74378f45a815779f19897191dff7d9efba9976b9e"
	# renovate-local: opentofu-darwin-amd64
	OPENTOFU_SHA256_DARWIN_AMD64="44bb1855f372f17f365fb94517906e78da5001da10f4c98de57a39bf982f3a92"
	# renovate-local: opentofu-darwin-arm64
	OPENTOFU_SHA256_DARWIN_ARM64="f958ec5e511063be9feb180ca015a4cb7977566a9cf6a8550bba8c2a9b5aba74"
	curl --output "${OPENTOFU_ARCHIVE}" --location --fail "${OPENTOFU_URL}"
	touch "${OPENTOFU_SHA256SUMS}"
	cat > "${OPENTOFU_SHA256SUMS}" <<-EOF
	LINUX_AMD64  ${OPENTOFU_SHA256_LINUX_AMD64}
	LINUX_ARM64  ${OPENTOFU_SHA256_LINUX_ARM64}
	DARWIN_AMD64 ${OPENTOFU_SHA256_DARWIN_AMD64}
	DARWIN_ARM64 ${OPENTOFU_SHA256_DARWIN_ARM64}
	EOF
	OPENTOFU_SHA256="$(set_sha_by_arch "${OPENTOFU_SHA256SUMS}")"
	OPENTOFU_SHA256_FILE="${OPENTOFU_FILENAME}.sha256"
	touch "${OPENTOFU_SHA256_FILE}"
	echo "${OPENTOFU_SHA256}" > "${OPENTOFU_SHA256_FILE}"
	verify_sha256_digest "${OPENTOFU_ARCHIVE}" "${OPENTOFU_SHA256_FILE}"
	rm -f "${OPENTOFU_SHA256SUMS}"
	rm -f "${OPENTOFU_SHA256_FILE}"
	mkdir "${OPENTOFU_FILENAME}"
	tar -xf "${OPENTOFU_ARCHIVE}" -C "${OPENTOFU_FILENAME}"
	mv "${OPENTOFU_FILENAME}/tofu" .
	rm -rf "${OPENTOFU_FILENAME}"*
}


vendor_kubectl() {
	echo "Downloading kubectl..."
	if [[ "${GOARCH}" != "amd64" && "${GOARCH}" != "arm64" ]]; then
		echo "Unsupported kubectl architecture for checksum validation: ${GOARCH}" >&2
		exit 1
	fi

	KUBECTL_BINARY="kubectl"

	KUBECTL_URL="https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/${GOOS}/${GOARCH}/${KUBECTL_BINARY}"
	KUBECTL_SHA256_SUMS="${KUBECTL_BINARY}_SHA256SUMS"
	touch "${KUBECTL_SHA256_SUMS}"
	# renovate-local: kubectl-linux-amd64
	KUBECTL_SHA256_LINUX_AMD64="6129359f4e1f3848a5572ccb0b26cf28b8ca08cef38c95a765b2f64a2c961a2f"
	# renovate-local: kubectl-linux-arm64
	KUBECTL_SHA256_LINUX_ARM64="922df28df248cc00a9e025f947704f1d1482de64ece54cfe57e61f19eaf1eef3"
	# renovate-local: kubectl-darwin-amd64
	KUBECTL_SHA256_DARWIN_AMD64="d5276c0f4fde77fc446070290f345944a7f1fda153df6b960e5fde93b7a9bccd"
	# renovate-local: kubectl-darwin-arm64
	KUBECTL_SHA256_DARWIN_ARM64="583beedaebe422e71d3f1a96acef8b1fef86ea2f09a45ad01aa6c9ce287c1380"
	cat > "${KUBECTL_SHA256_SUMS}" <<-EOF
	LINUX_AMD64  ${KUBECTL_SHA256_LINUX_AMD64}
	LINUX_ARM64  ${KUBECTL_SHA256_LINUX_ARM64}
	DARWIN_AMD64 ${KUBECTL_SHA256_DARWIN_AMD64}
	DARWIN_ARM64 ${KUBECTL_SHA256_DARWIN_ARM64}
	EOF
	KUBECTL_SHA256="$(set_sha_by_arch "${KUBECTL_SHA256_SUMS}")"
	KUBECTL_SHA256_FILE="${KUBECTL_BINARY}.sha256"
	touch "${KUBECTL_SHA256_FILE}"
	echo "${KUBECTL_SHA256}" > "${KUBECTL_SHA256_FILE}"
	curl --output "${KUBECTL_BINARY}" --location --fail "${KUBECTL_URL}"
	verify_sha256_digest "${KUBECTL_BINARY}" "${KUBECTL_SHA256_FILE}"
	chmod +x "${KUBECTL_BINARY}"
	rm -f "${KUBECTL_SHA256_FILE}"
	rm -f "${KUBECTL_SHA256_SUMS}"
}

vendor_helm() {
	echo Downloading and unpacking Helm...
	HELM_FILENAME="helm-v${HELM_VERSION}-${GOOS}-${GOARCH}"
	HELM_ARCHIVE="${HELM_FILENAME}.tar.gz"
	HELM_URL="https://get.helm.sh/${HELM_ARCHIVE}"
	HELM_SHA256_SUMS="${HELM_ARCHIVE}_SHA256SUMS"
	touch "${HELM_SHA256_SUMS}"
	# renovate-local: helm-linux-amd64
	HELM_SHA256_LINUX_AMD64="86584a54def73570558f66f5111cc53dfed56689637ae32c1201205d494f54fb"
	# renovate-local: helm-linux-arm64
	HELM_SHA256_LINUX_ARM64="31c5794dd55c66a51e6b7d2e2ac7a114ae8b1de41ff1d9ba51748ac973b06a08"
	# renovate-local: helm-darwin-amd64
	HELM_SHA256_DARWIN_AMD64="347a784877e0e20eac865e8d1c36a80f6bb0861d6f29abd34defb6570ef95d92"
	# renovate-local: helm-darwin-arm64
	HELM_SHA256_DARWIN_ARM64="d3870437e1e95b67f8edbde964156c84a26503f560821d40c542441658934fba"
	cat > "${HELM_SHA256_SUMS}" <<-EOF
	LINUX_AMD64  ${HELM_SHA256_LINUX_AMD64}
	LINUX_ARM64  ${HELM_SHA256_LINUX_ARM64}
	DARWIN_AMD64 ${HELM_SHA256_DARWIN_AMD64}
	DARWIN_ARM64 ${HELM_SHA256_DARWIN_ARM64}
	EOF
	HELM_SHA256="$(set_sha_by_arch "${HELM_SHA256_SUMS}")"
	HELM_SHA256_FILE="${HELM_FILENAME}.sha256"
	touch "${HELM_SHA256_FILE}"
	echo "${HELM_SHA256}" > "${HELM_SHA256_FILE}"
	curl --output "${HELM_ARCHIVE}" --location --fail "${HELM_URL}"
	verify_sha256_digest "${HELM_ARCHIVE}" "${HELM_SHA256_FILE}"
	rm -f "${HELM_SHA256_FILE}"
	rm -f "${HELM_SHA256_SUMS}"
	tar xvf "${HELM_ARCHIVE}"
	mv "${GOOS}-${GOARCH}"/helm .
	rm -rf "${HELM_ARCHIVE}"*
	rm -rf "${GOOS}"-"${GOARCH}"*
}

vendor_k3d() {
	echo Downloading k3d...
	K3D_BINARY="k3d-${GOOS}-${GOARCH}"
	K3D_URL="https://github.com/k3d-io/k3d/releases/download/v${K3D_VERSION}/${K3D_BINARY}"
	K3D_CHECKSUMS="${K3D_BINARY}_SHA256SUMS"
	touch "${K3D_CHECKSUMS}"
	# renovate-local: k3d-linux-amd64
	K3D_SHA256_LINUX_AMD64="06d8f25bc3a971c4eb29e0ff08429b180402db0f4dec838c9eac427e296800a0"
	# renovate-local: k3d-linux-arm64
	K3D_SHA256_LINUX_ARM64="03cde5cf23e6e8e67de5a039ecf26e5b85aca82fba3e5d13dadf904cd218a250"
	# renovate-local: k3d-darwin-amd64
	K3D_SHA256_DARWIN_AMD64="b4aabc37534f95b9c764e7823f2df923f50d57600837aa60a06266cce47db732"
	# renovate-local: k3d-darwin-arm64
	K3D_SHA256_DARWIN_ARM64="fe106541d5d0a3f18debcd4d432a16f8c0ce3e6ddc06f8fbb6f696a122313e00"
	cat > "${K3D_CHECKSUMS}" <<-EOF
	LINUX_AMD64  ${K3D_SHA256_LINUX_AMD64}
	LINUX_ARM64  ${K3D_SHA256_LINUX_ARM64}
	DARWIN_AMD64 ${K3D_SHA256_DARWIN_AMD64}
	DARWIN_ARM64 ${K3D_SHA256_DARWIN_ARM64}
	EOF
	K3D_SHA256="$(set_sha_by_arch "${K3D_CHECKSUMS}")"
	K3D_SHA256_FILE="${K3D_BINARY}.sha256"
	touch "${K3D_SHA256_FILE}"
	echo "${K3D_SHA256}" > "${K3D_SHA256_FILE}"
	curl --output "${K3D_BINARY}" --location --fail "${K3D_URL}"
	verify_sha256_digest "${K3D_BINARY}" "${K3D_SHA256_FILE}"
	chmod +x "${K3D_BINARY}"
	rm -f "${K3D_SHA256_FILE}"
	rm -f "${K3D_CHECKSUMS}"
	mv "${K3D_BINARY}" k3d
}

rm -rf internal/vendored/bin
mkdir -p internal/vendored/bin
cd internal/vendored/bin
export PATH="$(pwd):${PATH}"

vendor_opentofu
vendor_kubectl
vendor_helm
vendor_k3d