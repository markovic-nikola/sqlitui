#!/bin/sh
set -e

REPO="markovic-nikola/sqlitui"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS.
OS="$(uname -s)"
case "$OS" in
  Linux*)  OS="linux" ;;
  Darwin*) OS="darwin" ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture.
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Get latest version tag.
VERSION="${VERSION:-$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)}"
if [ -z "$VERSION" ]; then
  echo "Failed to fetch latest version" >&2
  exit 1
fi

ARCHIVE="sqlitui_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading sqlitui ${VERSION} (${OS}/${ARCH})..."
curl -sSfL "${BASE_URL}/${ARCHIVE}" -o "${TMP_DIR}/${ARCHIVE}"
curl -sSfL "${BASE_URL}/checksums.txt" -o "${TMP_DIR}/checksums.txt"

# Verify checksum.
EXPECTED="$(grep "${ARCHIVE}" "${TMP_DIR}/checksums.txt" | awk '{print $1}')"
if [ -z "$EXPECTED" ]; then
  echo "Checksum not found for ${ARCHIVE}" >&2
  exit 1
fi

ACTUAL="$(sha256sum "${TMP_DIR}/${ARCHIVE}" | awk '{print $1}')"
if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "Checksum verification failed" >&2
  echo "  expected: ${EXPECTED}" >&2
  echo "  actual:   ${ACTUAL}" >&2
  exit 1
fi

echo "Checksum verified."

# Extract and install.
tar xzf "${TMP_DIR}/${ARCHIVE}" -C "${TMP_DIR}"

if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP_DIR}/sqlitui" "${INSTALL_DIR}/sqlitui"
else
  sudo mv "${TMP_DIR}/sqlitui" "${INSTALL_DIR}/sqlitui"
fi

echo "Installed sqlitui to ${INSTALL_DIR}/sqlitui"
