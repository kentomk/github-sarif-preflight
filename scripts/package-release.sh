#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  printf '%s\n' 'usage: scripts/package-release.sh VERSION OUTPUT_DIRECTORY' >&2
  exit 2
fi

version=$1
output_directory=$2
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  printf 'invalid semantic version: %s\n' "$version" >&2
  exit 2
fi

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
mkdir -p "$output_directory"
output_directory=$(cd "$output_directory" && pwd -P)
build_root=$(mktemp -d "${TMPDIR:-/tmp}/github-sarif-preflight-release.XXXXXX")
trap 'rm -rf -- "$build_root"' EXIT

source_date_epoch=${SOURCE_DATE_EPOCH:-0}
targets=(
  linux/amd64
  linux/arm64
  darwin/amd64
  darwin/arm64
)
archives=()

for target in "${targets[@]}"; do
  target_os=${target%/*}
  target_arch=${target#*/}
  archive_root="github-sarif-preflight_${version}_${target_os}_${target_arch}"
  package_directory="$build_root/$archive_root"
  mkdir -p "$package_directory"

  (
    cd "$repository_root"
    CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" \
      go build -trimpath -buildvcs=false \
      -ldflags="-buildid= -s -w -X main.version=$version" \
      -o "$package_directory/github-sarif-preflight" \
      ./cmd/github-sarif-preflight
  )
  cp "$repository_root/README.md" "$repository_root/LICENSE" "$repository_root/SECURITY.md" "$package_directory/"
  chmod 0755 "$package_directory/github-sarif-preflight"
  chmod 0644 "$package_directory/README.md" "$package_directory/LICENSE" "$package_directory/SECURITY.md"

  archive="$output_directory/$archive_root.tar.gz"
  tar --sort=name --format=ustar --owner=0 --group=0 --numeric-owner \
    --mtime="@$source_date_epoch" -C "$build_root" -cf - "$archive_root" |
    gzip -n -9 >"$archive"
  archives+=("${archive##*/}")
done

(
  cd "$output_directory"
  sha256sum "${archives[@]}" >SHA256SUMS
)

printf 'packaged %s archives for %s in %s\n' "${#archives[@]}" "$version" "$output_directory"
