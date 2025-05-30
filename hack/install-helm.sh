#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

HELM_VERSION="v3.12.0"
ARCHIVE="helm-${HELM_VERSION}-linux-amd64.tar.gz"

echo "Downloading Helm ${HELM_VERSION}..."
curl -fsSL "https://get.helm.sh/${ARCHIVE}" -o "${ARCHIVE}"

echo "Unpacking..."
tar -xzf "${ARCHIVE}"

echo "Installing to /usr/local/bin..."
sudo mv linux-amd64/helm /usr/local/bin/helm

echo "Cleaning up..."
rm -rf "${ARCHIVE}" linux-amd64

echo "Helm ${HELM_VERSION} installed successfully."
