#!/usr/bin/env bash
set -euo pipefail

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
cd "$repository_root"

if [[ $(go list -m all | wc -l) -ne 1 ]]; then
  printf '%s\n' 'runtime module graph must contain only this module' >&2
  go list -m all >&2
  exit 1
fi

runtime_imports=$(go list -f '{{range .Imports}}{{println .}}{{end}}' ./cmd/... ./internal/... | sort -u)
for forbidden in net net/http net/rpc os/exec plugin; do
  if grep -Fxq "$forbidden" <<<"$runtime_imports"; then
    printf 'forbidden runtime capability import: %s\n' "$forbidden" >&2
    exit 1
  fi
done

if ! grep -q 'Apache License' LICENSE; then
  printf '%s\n' 'Apache-2.0 license text is missing' >&2
  exit 1
fi

if git grep -nIE '(-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----|github_pat_[A-Za-z0-9_]{20,}|gh[pousr]_[A-Za-z0-9]{36,}|AKIA[0-9A-Z]{16})' -- . \
  ':!scripts/test-policy.sh'; then
  printf '%s\n' 'secret-like tracked content detected' >&2
  exit 1
fi

if rg -n 'uses:[[:space:]]+[^[:space:]]+@(main|master|v[0-9]+)([[:space:]]|$)' .github action.yml; then
  printf '%s\n' 'GitHub Action dependency is not pinned to an immutable revision' >&2
  exit 1
fi

printf '%s\n' 'policy passed: external-modules=0 forbidden-runtime-imports=0 secret-patterns=0 action-pins=immutable license=Apache-2.0'
