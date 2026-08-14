#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

forge soldeer install
(cd lib/flare-foundry-periphery-package && forge soldeer install)

bash ./scripts/generate-bindings.sh
bash ./scripts/generate-workproof-bindings.sh
bash ./scripts/generate-workproof-tools-bindings.sh

echo "bootstrap complete"
