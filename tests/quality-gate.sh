#!/usr/bin/env bash
set -euo pipefail

project_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
cd "$project_root"

test -z "$(gofmt -l cmd internal)"
go test ./...
go test -race ./...
go vet ./...
find scripts tests -type f -name '*.sh' -print0 | sort -z | xargs -0 -n1 bash -n
scripts/test-policy.sh
binary=$(mktemp "${TMPDIR:-/tmp}/github-sarif-preflight-quality.XXXXXX")
trap 'rm -f -- "$binary"' EXIT
go build -trimpath -buildvcs=false -o "$binary" ./cmd/github-sarif-preflight
scripts/test-performance.sh "$binary"
scripts/test-release.sh
printf '%s\n' 'quality gate passed'
