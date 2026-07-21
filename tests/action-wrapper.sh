#!/usr/bin/env bash
set -euo pipefail

project_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
test_root=$(mktemp -d "${TMPDIR:-/tmp}/github-sarif-preflight-action-review.XXXXXX")
trap 'rm -rf -- "$test_root"' EXIT

export GITHUB_ACTION_PATH="$project_root"
export RUNNER_TEMP="$test_root"

GSP_ACTION_ROOT="$project_root/testdata/safe-srcroot" \
GSP_ACTION_SARIF="$project_root/testdata/safe-srcroot/results.sarif" \
  "$project_root/scripts/run-action.sh" >/dev/null

set +e
output=$(GSP_ACTION_ROOT="$project_root/testdata/missing-inline-message" \
  GSP_ACTION_SARIF="$project_root/testdata/missing-inline-message/results.sarif" \
  "$project_root/scripts/run-action.sh" 2>&1)
status=$?
set -e
[[ $status -eq 1 && $output == *GSP001* ]]

set +e
GSP_ACTION_ROOT="$project_root/testdata/invalid-json" \
GSP_ACTION_SARIF="$project_root/testdata/invalid-json/results.sarif" \
  "$project_root/scripts/run-action.sh" >/dev/null 2>&1
status=$?
set -e
[[ $status -eq 2 ]]

printf '%s\n' 'action wrapper passed: safe=0 diagnostic=1 invalid=2 token-required=false'
