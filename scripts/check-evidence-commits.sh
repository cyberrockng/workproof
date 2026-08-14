#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

files=(
  "$ROOT_DIR/deployments/coston2.json"
  "$ROOT_DIR/docs/evidence/demo-run.json"
)

for file in "${files[@]}"; do
  commit="$(jq -r '.sourceCommit // empty' "$file")"
  if [[ -z "$commit" ]]; then
    echo "missing sourceCommit in ${file#$ROOT_DIR/}" >&2
    exit 1
  fi
  if ! git -C "$ROOT_DIR" cat-file -e "$commit^{commit}" 2>/dev/null; then
    echo "sourceCommit $commit in ${file#$ROOT_DIR/} is not present in local git history" >&2
    exit 1
  fi
  if ! git -C "$ROOT_DIR" merge-base --is-ancestor "$commit" HEAD; then
    echo "sourceCommit $commit in ${file#$ROOT_DIR/} is not an ancestor of HEAD" >&2
    exit 1
  fi
  echo "sourceCommit ok: ${file#$ROOT_DIR/} -> $commit"
done
