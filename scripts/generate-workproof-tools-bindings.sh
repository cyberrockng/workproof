#!/usr/bin/env bash
# generate-workproof-tools-bindings.sh — Compile WorkProofEscrow.sol and
# generate full (Deploy+Transact+Call) Go bindings for the tools/ module,
# the same abigen recipe scripts/generate-bindings.sh uses for the
# scaffold's own HelloWorldInstructionSender (--abi AND --bin, so abigen
# emits a Deploy function -- unlike scripts/generate-workproof-bindings.sh,
# which only needs read bindings for the go/ extension module and
# deliberately omits --bin).
#
# Prerequisites: forge (Foundry), jq
#
# Usage: ./scripts/generate-workproof-tools-bindings.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

CONTRACT_NAME="WorkProofEscrow"
GO_PKG="workproofescrow"
BINDINGS_DIR="$PROJECT_DIR/tools/pkg/contracts/$GO_PKG"

cd "$PROJECT_DIR"

echo "=== Step 1: Compile Solidity contracts ==="
forge build

echo "=== Step 2: Extract ABI and BIN ==="
FORGE_OUT="$PROJECT_DIR/out/${CONTRACT_NAME}.sol/${CONTRACT_NAME}.json"
if [[ ! -f "$FORGE_OUT" ]]; then
    echo "ERROR: forge output not found at $FORGE_OUT"
    exit 1
fi

mkdir -p "$BINDINGS_DIR"
jq '.abi' "$FORGE_OUT" > "$BINDINGS_DIR/${CONTRACT_NAME}.abi"
jq -r '.bytecode.object' "$FORGE_OUT" | sed 's/^0x//' > "$BINDINGS_DIR/${CONTRACT_NAME}.bin"

echo "  ABI -> $BINDINGS_DIR/${CONTRACT_NAME}.abi"
echo "  BIN -> $BINDINGS_DIR/${CONTRACT_NAME}.bin"

echo "=== Step 3: Generate Go bindings ==="
cd "$PROJECT_DIR/tools"
go generate ./pkg/contracts/$GO_PKG/

echo "=== Done ==="
echo "Generated: $BINDINGS_DIR/autogen.go"
