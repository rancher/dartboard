#!/usr/bin/env bash

set -e

OPENTOFU_VERSION=1.8.2
KUBECTL_VERSION=1.31.1
HELM_VERSION=3.16.1
K6_VERSION=0.54.0
K3D_VERSION=5.7.4

GOOS=`go env GOOS`
GOARCH=`go env GOARCH`

rm -rf internal/vendored/bin
mkdir -p internal/vendored/bin
cd internal/vendored/bin


echo Downloading and unpacking OpenTofu...
OPENTOFU_FILENAME="tofu_${OPENTOFU_VERSION}_${GOOS}_${GOARCH}"
OPENTOFU_ARCHIVE="${OPENTOFU_FILENAME}.zip"
OPENTOFU_URL="https://github.com/opentofu/opentofu/releases/download/v${OPENTOFU_VERSION}/${OPENTOFU_ARCHIVE}"
curl --output ${OPENTOFU_ARCHIVE} --location --fail ${OPENTOFU_URL}
mkdir ${OPENTOFU_FILENAME}
unzip ${OPENTOFU_ARCHIVE} -d ${OPENTOFU_FILENAME}
mv ${OPENTOFU_FILENAME}/tofu .
rm -rf ${OPENTOFU_FILENAME}*

echo Downloading kubectl...
KUBECTL_URL="https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/${GOOS}/${GOARCH}/kubectl"
curl --output kubectl --location --fail ${KUBECTL_URL}

echo Downloading and unpacking Helm...
HELM_FILENAME="helm-v${HELM_VERSION}-${GOOS}-${GOARCH}"
HELM_ARCHIVE="${HELM_FILENAME}.tar.gz"
HELM_URL="https://get.helm.sh/${HELM_ARCHIVE}"
curl --output ${HELM_ARCHIVE} --location --fail ${HELM_URL}
tar xvf ${HELM_ARCHIVE}
mv ${GOOS}-${GOARCH}/helm .
rm -rf ${HELM_ARCHIVE}*
rm -rf ${GOOS}-${GOARCH}*

echo Downloading k3d...
K3D_URL="https://github.com/k3d-io/k3d/releases/download/v${K3D_VERSION}/k3d-${GOOS}-${GOARCH}"
curl --output k3d --location --fail ${K3D_URL}
