#!/usr/bin/env bash
set -euo pipefail

project_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
test_root=$(mktemp -d "${TMPDIR:-/tmp}/github-sarif-preflight-clean-review.XXXXXX")
trap 'rm -rf -- "$test_root"' EXIT

SOURCE_DATE_EPOCH=0 "$project_root/scripts/package-release.sh" v0.1.0 "$test_root/dist" >/dev/null
(
  cd "$test_root/dist"
  sha256sum -c SHA256SUMS >/dev/null
)
tar -xzf "$test_root/dist/github-sarif-preflight_v0.1.0_linux_arm64.tar.gz" -C "$test_root"
package_root="$test_root/github-sarif-preflight_v0.1.0_linux_arm64"
mkdir -p "$test_root/checkout"
printf '%s\n' '{"version":"2.1.0","runs":[{"results":[{"ruleId":"demo","message":{},"locations":[]}]}]}' >"$test_root/results.sarif"

start=$(date +%s)
set +e
output=$("$package_root/github-sarif-preflight" check --root "$test_root/checkout" "$test_root/results.sarif" 2>&1)
status=$?
set -e
elapsed=$(( $(date +%s) - start ))
[[ $status -eq 1 && $output == *GSP001* && $elapsed -lt 300 ]]
! grep -Fq "$test_root/checkout" <<<"$output"
printf 'clean archive quickstart passed: first-useful-output=%ss exit=1 diagnostic=GSP001\n' "$elapsed"
