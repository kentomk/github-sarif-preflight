#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
  printf '%s\n' 'usage: scripts/test-alternatives.sh PREFLIGHT_BINARY SARIF_MULTITOOL_COMMAND...' >&2
  exit 2
fi

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
preflight_binary=$1
shift
multitool=("$@")

if [[ ! -x "$preflight_binary" ]]; then
  printf 'preflight binary is not executable: %s\n' "$preflight_binary" >&2
  exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
  printf '%s\n' 'jq is required for the pinned shape comparison' >&2
  exit 2
fi

multitool_version=$("${multitool[@]}" --version 2>&1)
if [[ "$multitool_version" != *"5.5.0"* ]]; then
  printf 'expected Sarif.Multitool 5.5.0, got: %s\n' "$multitool_version" >&2
  exit 2
fi

fixtures=(
  'missing-inline-message:GSP001'
  'empty-artifact-uri:GSP002'
  'unsupported-base-id:GSP003'
  'root-escape:GSP004'
)

for pair in "${fixtures[@]}"; do
  fixture=${pair%%:*}
  expected_id=${pair##*:}
  sarif_file="$repository_root/testdata/$fixture/results.sarif"

  jq -e '.version == "2.1.0" and (.runs | type == "array")' "$sarif_file" >/dev/null

  set +e
  multitool_output=$("${multitool[@]}" validate "$sarif_file" 2>&1)
  multitool_status=$?
  set -e
  if [[ $multitool_status -ne 0 ]] || grep -q ': error ' <<<"$multitool_output"; then
    printf 'Sarif.Multitool unexpectedly rejected %s\n%s\n' "$fixture" "$multitool_output" >&2
    exit 1
  fi

  set +e
  preflight_output=$("$preflight_binary" check --root "$repository_root/testdata/$fixture" "$sarif_file" 2>&1)
  preflight_status=$?
  set -e
  if [[ $preflight_status -ne 1 ]] || ! grep -q "$expected_id" <<<"$preflight_output"; then
    printf 'preflight did not produce %s for %s\n%s\n' "$expected_id" "$fixture" "$preflight_output" >&2
    exit 1
  fi
done

printf '%s\n' 'alternative regression passed: Sarif.Multitool=5.5.0 jq-shape=pass preflight=GSP001..GSP004'
