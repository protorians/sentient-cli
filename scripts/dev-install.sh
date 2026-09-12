#!/usr/bin/env bash
#
# dev-install.sh — Compile et installe le binaire `sentients` (mode dev)
# pour qu'il soit disponible dans toute la session machine (PATH global).
#
# Usage:
#   ./scripts/dev-install.sh [--prefix=PATH] [--force]
#
# Comportement:
#   - Installe dans /usr/local/bin si disponible, sinon ~/.local/bin.
#   - Ajoute le dossier destination au PATH de la session courante.
#   - Désinstalle l'ancien binaire via ./scripts/dev-uninstall.sh si présent.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PREFIX=""
FORCE=false
for arg in "$@"; do
  case "$arg" in
    --prefix=*) PREFIX="${arg#--prefix=}" ;;
    --force) FORCE=true ;;
    *) echo "Usage: $0 [--prefix=PATH] [--force]" >&2; exit 1 ;;
  esac
done

GO="$(command -v go || true)"
if [ -z "$GO" ]; then
  echo "Error: 'go' is required to build the CLI in dev mode." >&2
  exit 1
fi

BINARY="sentients"
TMP_DIR="$(mktemp -d -t sentients.XXXXXX)"
TMP_BIN="$TMP_DIR/$BINARY"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Building $BINARY (dev)..."
GOFLAGS=-mod=mod CGO_ENABLED=0 go build \
  -ldflags "-X main.version=dev -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo none) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o "$TMP_BIN" .

if [ -n "$PREFIX" ]; then
  INSTALL_DIR="$PREFIX"
else
  if [ -w /usr/local/bin ]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="${HOME}/.local/bin"
  fi
fi

mkdir -p "$INSTALL_DIR"

if [ -e "$INSTALL_DIR/$BINARY" ] && [ "$FORCE" != true ]; then
  echo "Warning: $INSTALL_DIR/$BINARY already exists. Skipping overwrite (use --force)." >&2
  echo "Existing binary: $("$INSTALL_DIR/$BINARY" --version 2>/dev/null || echo unknown)" >&2
  exit 1
fi

install -m 0755 "$TMP_BIN" "$INSTALL_DIR/$BINARY"

# PATH de la session courante
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) export PATH="$INSTALL_DIR:$PATH" ;;
esac

echo ""
echo "Installed: $INSTALL_DIR/$BINARY"
"$INSTALL_DIR/$BINARY" --version
echo "Available in this session as: sentients"

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo ""
  echo "Add $INSTALL_DIR to your PATH to use it in every session:"
  echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
fi