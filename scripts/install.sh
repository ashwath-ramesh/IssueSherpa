#!/usr/bin/env bash
set -euo pipefail

REPO="ashwath-ramesh/IssueSherpa"
BINARY="issuesherpa"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  darwin|linux) ;;
  *) echo "Error: unsupported OS $OS (supported: darwin, linux)." >&2; exit 1 ;;
esac

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  arm64)   ARCH="arm64" ;;
  aarch64) ARCH="arm64" ;;
  *) echo "Error: unsupported architecture $ARCH." >&2; exit 1 ;;
esac

REQUESTED_VERSION="${VERSION:-}"
if [[ -n "$REQUESTED_VERSION" ]]; then
  TAG="${REQUESTED_VERSION#v}"
else
  echo "Fetching latest release..."
  TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')
fi

if [[ -z "$TAG" ]]; then
  echo "Error: could not determine release version." >&2
  exit 1
fi

echo "Installing ${BINARY} v${TAG} (${OS}/${ARCH})..."

TARBALL="${BINARY}_${TAG}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${TAG}/${TARBALL}"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/v${TAG}/checksums.txt"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$URL" -o "${TMP}/${TARBALL}"
curl -fsSL "$CHECKSUMS_URL" -o "${TMP}/checksums.txt"

EXPECTED_HASH=$(awk -v file="$TARBALL" '{name=$2; sub(/^\*/, "", name); if (name == file) print tolower($1)}' "${TMP}/checksums.txt")
if [[ -z "$EXPECTED_HASH" ]]; then
  echo "Error: checksum not found for ${TARBALL}." >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL_HASH=$(sha256sum "${TMP}/${TARBALL}" | awk '{print tolower($1)}')
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL_HASH=$(shasum -a 256 "${TMP}/${TARBALL}" | awk '{print tolower($1)}')
else
  echo "Error: no SHA-256 tool found (need sha256sum or shasum)." >&2
  exit 1
fi

if [[ "$ACTUAL_HASH" != "$EXPECTED_HASH" ]]; then
  echo "Error: checksum verification failed for ${TARBALL}." >&2
  exit 1
fi

tar -xzf "${TMP}/${TARBALL}" -C "$TMP"

INSTALL_DIR="${INSTALL_DIR:-}"
if [[ -z "$INSTALL_DIR" ]]; then
  if [[ -w /usr/local/bin ]]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="${HOME}/.local/bin"
  fi
fi

mkdir -p "$INSTALL_DIR"
mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
chmod +x "${INSTALL_DIR}/${BINARY}"

if [[ "$OS" == "darwin" ]]; then
  xattr -d com.apple.quarantine "${INSTALL_DIR}/${BINARY}" 2>/dev/null || true
fi

echo ""
echo "Installed: ${INSTALL_DIR}/${BINARY} (v${TAG})"

if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  echo ""
  echo "Add to your PATH:"
  echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
fi

echo ""
echo "Next:"
echo "  issuesherpa init"
echo "  edit the generated config"
echo "  issuesherpa list"
