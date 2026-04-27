#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="$SCRIPT_DIR/../deploy-agent-bin"

GOTOOLCHAIN=local go build -o "$BIN" "$SCRIPT_DIR"
echo "Built deploy-agent"

sudo install -m 0755 "$BIN" /usr/local/bin/deploy-agent
rm "$BIN"
echo "Installed to /usr/local/bin/deploy-agent"

if systemctl is-active --quiet deploy-agent; then
    sudo systemctl restart deploy-agent
    echo "Restarted deploy-agent service"
fi
