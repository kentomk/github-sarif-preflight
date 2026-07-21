#!/usr/bin/env bash
set -euo pipefail

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
test_root=$(mktemp -d "${TMPDIR:-/tmp}/github-sarif-preflight-release-test.XXXXXX")
trap 'rm -rf -- "$test_root"' EXIT

version=v0.1.0-test.1
SOURCE_DATE_EPOCH=0 "$repository_root/scripts/package-release.sh" "$version" "$test_root/first"
SOURCE_DATE_EPOCH=0 "$repository_root/scripts/package-release.sh" "$version" "$test_root/second"

cmp "$test_root/first/SHA256SUMS" "$test_root/second/SHA256SUMS"
archive_count=$(find "$test_root/first" -maxdepth 1 -type f -name '*.tar.gz' | wc -l)
if [[ $archive_count -ne 4 ]]; then
  printf 'expected 4 release archives, found %s\n' "$archive_count" >&2
  exit 1
fi

while IFS= read -r archive; do
  name=${archive##*/}
  cmp "$archive" "$test_root/second/$name"
  entries=$(tar -tzf "$archive")
  for required in github-sarif-preflight README.md LICENSE SECURITY.md; do
    if ! grep -q "/$required$" <<<"$entries"; then
      printf '%s is missing %s\n' "$name" "$required" >&2
      exit 1
    fi
  done
done < <(find "$test_root/first" -maxdepth 1 -type f -name '*.tar.gz' | sort)

(
  cd "$test_root/first"
  sha256sum -c SHA256SUMS
)

case "$(uname -s)/$(uname -m)" in
  Linux/x86_64) host_target=linux_amd64 ;;
  Linux/aarch64 | Linux/arm64) host_target=linux_arm64 ;;
  Darwin/x86_64) host_target=darwin_amd64 ;;
  Darwin/arm64) host_target=darwin_arm64 ;;
  *) printf '%s\n' 'unsupported test host' >&2; exit 2 ;;
esac
host_archive="$test_root/first/github-sarif-preflight_${version}_${host_target}.tar.gz"
tar -xzf "$host_archive" -C "$test_root"
host_binary=$(find "$test_root/github-sarif-preflight_${version}_${host_target}" -maxdepth 1 -type f -name github-sarif-preflight -print -quit)
if [[ "$("$host_binary" version)" != "$version" ]]; then
  printf '%s\n' 'packaged binary version mismatch' >&2
  exit 1
fi

printf '%s\n' 'release packaging passed: targets=4 reproducible=true checksums=verified host-version=verified'
