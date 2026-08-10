#!/usr/bin/env bash
# generate-workproof-bindings.sh — Compile WorkProofEscrow.sol and generate
# Go read bindings for the extension module (go/), the same abigen recipe
# scripts/generate-bindings.sh uses for the scaffold's own
# HelloWorldInstructionSender, scoped to the workproof-specific contract and
# the go/ module instead of tools/.
#
# Prerequisites: forge (Foundry), jq
#
# Usage: ./scripts/generate-workproof-bindings.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

CONTRACT_NAME="WorkProofEscrow"
GO_PKG="workproofescrow"
BINDINGS_DIR="$PROJECT_DIR/go/pkg/contracts/$GO_PKG"

cd "$PROJECT_DIR"

echo "=== Step 1: Compile Solidity contracts ==="
forge build

echo "=== Step 2: Extract ABI ==="
FORGE_OUT="$PROJECT_DIR/out/${CONTRACT_NAME}.sol/${CONTRACT_NAME}.json"
if [[ ! -f "$FORGE_OUT" ]]; then
    echo "ERROR: forge output not found at $FORGE_OUT"
    exit 1
fi

mkdir -p "$BINDINGS_DIR"
jq '.abi' "$FORGE_OUT" > "$BINDINGS_DIR/${CONTRACT_NAME}.abi"
echo "  ABI -> $BINDINGS_DIR/${CONTRACT_NAME}.abi"

echo "=== Step 3: Generate Go bindings ==="
cd "$PROJECT_DIR/go"
go generate ./pkg/contracts/$GO_PKG/

echo "=== Done ==="
echo "Generated: $BINDINGS_DIR/autogen.go"
