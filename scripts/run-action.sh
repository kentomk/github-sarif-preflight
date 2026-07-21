#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GSP_ACTION_SARIF:-}" ]]; then
  printf '%s\n' 'action input sarif-file must not be empty' >&2
  exit 2
fi

action_binary=${GSP_ACTION_BINARY:-}
if [[ -z "$action_binary" ]]; then
  action_temp=$(mktemp -d "${RUNNER_TEMP:-/tmp}/github-sarif-preflight-action.XXXXXX")
  trap 'rm -rf -- "$action_temp"' EXIT
  action_binary="$action_temp/github-sarif-preflight"
  (
    cd "$GITHUB_ACTION_PATH"
    go build -trimpath -o "$action_binary" ./cmd/github-sarif-preflight
  )
fi

if [[ ! -x "$action_binary" ]]; then
  printf 'action binary is not executable: %s\n' "$action_binary" >&2
  exit 2
fi

exec "$action_binary" check \
  --root "${GSP_ACTION_ROOT:-.}" \
  --format "${GSP_ACTION_FORMAT:-text}" \
  "$GSP_ACTION_SARIF"
