#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]] || [[ ! -x $1 ]]; then
  printf '%s\n' 'usage: scripts/test-performance.sh PREFLIGHT_BINARY' >&2
  exit 2
fi

preflight_binary=$1
test_root=$(mktemp -d "${TMPDIR:-/tmp}/github-sarif-preflight-performance.XXXXXX")
trap 'rm -rf -- "$test_root"' EXIT
sarif_file="$test_root/results.sarif"

awk 'BEGIN {
  printf "{\"version\":\"2.1.0\",\"runs\":[{\"results\":["
  for (i = 0; i < 100000; i++) {
    if (i > 0) printf ","
    printf "{\"message\":{\"text\":\"ok\"}}"
  }
  print "]}]}"
}' >"$sarif_file"

metrics_file="$test_root/metrics"
/usr/bin/time -f '%e %M' -o "$metrics_file" \
  "$preflight_binary" check --root "$test_root" "$sarif_file" >/dev/null
read -r elapsed_seconds maximum_rss_kib <"$metrics_file"

awk -v elapsed="$elapsed_seconds" 'BEGIN { if (elapsed >= 30) exit 1 }'
if (( maximum_rss_kib >= 262144 )); then
  printf 'memory budget exceeded: %s KiB\n' "$maximum_rss_kib" >&2
  exit 1
fi

printf 'performance passed: results=100000 elapsed=%ss max-rss=%sKiB limits=30s/262144KiB\n' "$elapsed_seconds" "$maximum_rss_kib"
