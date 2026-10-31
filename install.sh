#!/bin/sh
set -eu

REPO_URL="https://github.com/fioneo/thunder.git"
BIN_DIR="${XDG_BIN_HOME:-$HOME/.local/bin}"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/thunder"
CONFIG_FILE="$CONFIG_DIR/config.json"

info() { printf '==> %s\n' "$1"; }
fail() { printf 'Error: %s\n' "$1" >&2; exit 1; }

# Check dependencies
for cmd in git go; do
    command -v "$cmd" >/dev/null 2>&1 || fail "'$cmd' is required but not installed"
done

# Temporary build directory, removed on exit
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

# Config: create the directory, then the file (only if it doesn't exist yet)
info "Preparing configuration in $CONFIG_DIR"
mkdir -p "$CONFIG_DIR"
if [ ! -f "$CONFIG_FILE" ]; then
    echo '{}' > "$CONFIG_FILE"
    info "Created $CONFIG_FILE"
else
    info "Existing config kept: $CONFIG_FILE"
fi

# Build
info "Cloning repository"
git clone --depth 1 "$REPO_URL" "$TMP_DIR/thunder"

info "Building"
(cd "$TMP_DIR/thunder" && go build -o thunder ./src)

# Install
mkdir -p "$BIN_DIR"
install -m 755 "$TMP_DIR/thunder/thunder" "$BIN_DIR/thunder"

info "Successfully installed to $BIN_DIR/thunder"

case ":$PATH:" in
    *":$BIN_DIR:"*) ;;
    *) printf 'Note: %s is not in your PATH. Add this to your shell profile:\n  export PATH="%s:$PATH"\n' "$BIN_DIR" "$BIN_DIR" ;;
esac
