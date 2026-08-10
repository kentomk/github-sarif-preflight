#!/usr/bin/env bash
# shellcheck disable=SC2016,SC2251
set -euo pipefail

project_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
cd "$project_root"

jq -e '
  .schemaVersion == 2 and (.action == "create" or .action == "update") and .owner == "kentomk" and
  .name == "github-sarif-preflight" and
  (.description | type == "string" and length >= 20 and length <= 160) and
  (.topics | type == "array" and length >= 1 and length <= 10 and index("kento-oss") != null) and
  .candidateId == "20260719T070904Z-7fe2" and
  (.targetUsers | length >= 10 and length <= 500) and
  (.jobToBeDone | length >= 10 and length <= 1000) and
  (.distributionPath | length >= 10 and length <= 500) and
  (.successMetric | length >= 10 and length <= 500) and
  .reviewAfterDays == 1 and .opportunityScore == 76 and
  (.demandEvidence | type == "array" and length >= 3 and
    all((.url | startswith("https://")) and (.kind | test("^[a-z][a-z0-9-]{2,49}$")) and (.independenceKey | length >= 3))) and
  ((.demandEvidence | map(.independenceKey | ascii_downcase) | unique | length) >= 3) and
  ((.demandEvidence | map(.kind) | unique | length) >= 2) and
  (.alternatives | type == "array" and length >= 3 and
    all((.url | startswith("https://")) and .tested == true and (.gap | length >= 10))) and
  .duplicateSearch.completed == true and (.duplicateSearch.summary | length >= 20) and
  (.differentiation | length >= 20) and
  .testCommand == "scripts/publisher-gate.sh" and .license == "Apache-2.0" and
  (.commitMessage | length >= 10 and length <= 120)
' publish-request.json >/dev/null

jq -e --slurpfile request publish-request.json '
  .schemaVersion == 1 and .candidateId == $request[0].candidateId and
  .owner == "kentomk" and .author == "@kentomk" and
  .automatedAgent == true and
  (.createdBy | test("Matsuki Kento") and test("@kentomk") and test("AI|automated"; "i"))
' .kento-oss.json >/dev/null

grep -Eq '^## (Installation|Install|Getting Started)\b' README.md
grep -Eq '^## Quick[[:space:]]*start\b' README.md
grep -q 'Matsuki Kento' README.md
grep -q '@kentomk' README.md
grep -Eiq 'AI|automated' README.md
grep -q '^## Use this when$' README.md
grep -q '^## Do not use this when$' README.md
grep -Fq 'passes generic schema validation but GitHub Code Scanning rejects the upload' README.md
grep -Fq 'general SARIF schema validator' README.md
grep -Eq 'uses: actions/checkout@[0-9a-f]{40}([[:space:]]|$)' .github/workflows/ci.yml
grep -Eq 'uses: actions/setup-go@[0-9a-f]{40}([[:space:]]|$)' .github/workflows/ci.yml
! grep -Eq 'uses: actions/(checkout|setup-go)@v[0-9]' .github/workflows/ci.yml
grep -Eq '^- uses: actions/setup-go@[0-9a-f]{40}([[:space:]]|$)' README.md
grep -Eq '^- uses: kentomk/github-sarif-preflight@[0-9a-f]{40}([[:space:]]|$)' README.md
! grep -Eq '^- uses: (actions/setup-go|kentomk/github-sarif-preflight)@v[0-9]' README.md
! grep -q '<immutable-commit-sha>' README.md
grep -Fq 'github-sarif-preflight@v0.1.3' README.md
grep -Fq 'releases/tag/v0.1.3' README.md
grep -Fq 'sha256sum --check --strict -' README.md
grep -Fq 'curl -fsSLo SHA256SUMS' README.md
grep -Fq 'checksum_matches=$(grep -Ec' README.md
grep -Fq 'test "$checksum_matches" -eq 1' README.md
grep -Fq 'grep -E "^[0-9a-fA-F]{64}  $archive$" SHA256SUMS' README.md
grep -Fq 'unsafe_member=$(tar -tzf "$archive" | grep -E' README.md
grep -Fq 'extract_dir=$(mktemp -d)' README.md
grep -Fq 'tar -xzf "$archive" -C "$extract_dir"' README.md
grep -Fq 'expected_binary="$extract_dir/github-sarif-preflight_v0.1.3_linux_amd64/github-sarif-preflight"' README.md
grep -Fq 'test -f "$expected_binary" && test ! -L "$expected_binary"' README.md
grep -Fq 'mkdir -p "$HOME/.local/bin"' README.md
grep -Fq 'install -m 0755 "$expected_binary"' README.md
grep -Fq 'mv -f "$HOME/.local/bin/github-sarif-preflight.new"' README.md
grep -Fq 'github-sarif-preflight@f4728fec9562b8c1a77ea3a47fd689b025b1a58d # v0.1.3 release revision' README.md
grep -Fq 'package-release.sh" v0.1.3' tests/quickstart-clean.sh
grep -Fq 'github-sarif-preflight_v0.1.3_linux_arm64.tar.gz' tests/quickstart-clean.sh
grep -Fq 'github-sarif-preflight_v0.1.3_linux_arm64' tests/quickstart-clean.sh
! grep -Eq 'package-release.sh" v0.1\.[012]|github-sarif-preflight_v0.1\.([012])' tests/quickstart-clean.sh
! grep -Eq 'github-sarif-preflight@v0.1.[12]|package-release.sh v0.1.[12]|The `v0.1.[12]` release' README.md
grep -Fq 'The published' SECURITY.md
grep -Fq 'v0.1.3' SECURITY.md
if grep -Fq 'No public release exists yet' SECURITY.md; then
  echo 'SECURITY.md still claims the public project is unpublished' >&2
  exit 1
fi
