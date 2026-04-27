#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

GOTOOLCHAIN=local go build -o /usr/local/bin/deploy-agent "$SCRIPT_DIR"
echo "Built and installed deploy-agent to /usr/local/bin/deploy-agent"

if systemctl is-active --quiet deploy-agent; then
    sudo systemctl restart deploy-agent
    echo "Restarted deploy-agent service"
fi
