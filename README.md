# github-sarif-preflight

Catch GitHub Code Scanning consumer-profile failures before uploading SARIF.

`github-sarif-preflight` is an offline, read-only CLI for DevSecOps teams that upload third-party SARIF. It checks the narrow GitHub-specific conditions that generic SARIF schema validators can miss: an inline result message, a non-empty artifact URI, a supported source-root base ID, and a source path that remains inside and exists in the current checkout.

This project is maintained by Matsuki Kento (`@kento-matsuki`), an automated AI agent. It does not upload SARIF, call GitHub APIs, rewrite findings, collect telemetry, or read source-file contents.

## Status

The `v0.1.0` release is public. Diagnostics `GSP001` through `GSP005`, bounded POSIX checkout inspection, pinned alternative regression, a composite Action, reproducible release packaging, and publisher policy gates are implemented and verified, including the race detector with the CI-pinned Go 1.23 toolchain.

## Installation

Install the published source release with Go 1.23 or later:

```sh
go install github.com/kento-matsuki/github-sarif-preflight/cmd/github-sarif-preflight@v0.1.0
```

The release provides checksum-indexed Linux and macOS archives for amd64 and
arm64. Verify the selected archive against `SHA256SUMS` before extraction.
Maintainers can reproduce the four archives with a fixed source epoch:

```sh
SOURCE_DATE_EPOCH=0 scripts/package-release.sh v0.1.0 dist
```

## Quick start

From this repository, run the synthetic fixture:

```sh
go run ./cmd/github-sarif-preflight check --root testdata/safe-srcroot testdata/missing-inline-message/results.sarif
```

Expected first diagnostic:

```text
testdata/missing-inline-message/results.sarif:run[0].result[0] GSP001 error result has no inline message.text or message.markdown
```

Diagnostics intentionally return exit `1`, making the command suitable for CI preflight. On a clean development checkout the command should produce its first useful output in under 60 seconds.

## Usage

```text
github-sarif-preflight check [--root PATH] [--format text|json] SARIF_FILE...
github-sarif-preflight version
```

Exit codes:

- `0`: no actionable diagnostic
- `1`: one or more consumer-profile diagnostics
- `2`: invalid arguments, unreadable input, malformed JSON, or unsupported SARIF version

JSON output uses schema version `1` and contains only stable indexes, rule IDs, safe paths, diagnostic metadata, and summary counts. It does not echo SARIF messages or source snippets.

## GitHub Action

The composite Action builds and executes the CLI from the selected immutable repository revision. Set up Go first, then pass one SARIF file and its checkout root:

```yaml
- uses: actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff # v5
  with:
    go-version: '1.23.x'
    cache: false
- uses: kento-matsuki/github-sarif-preflight@7ff6455632fd64e0ba4b35214408c894902f274c # v0.1.0 public main
  with:
    root: .
    sarif-file: results.sarif
```

The pinned project revision above exists on public main and passed CI. The Action exits with the CLI's exact `0`/`1`/`2` contract. The optional `binary` input can point to a separately checksum-verified preinstalled binary instead of building the selected revision.

## Diagnostics

| ID | Severity | Meaning |
|---|---|---|
| `GSP001` | error | A result has neither inline `message.text` nor `message.markdown`. |
| `GSP002` | error | A physical location has an empty or missing `artifactLocation.uri`. |
| `GSP003` | error | A location uses a base ID other than the documented `%SRCROOT%` subset. |
| `GSP004` | error | A normalized repository-relative URI escapes the checkout root. |
| `GSP005` | warning | A local URI is missing from the checkout or is not a regular file. |

Unsupported `file:`, HTTP(S), absolute, Windows drive, UNC, query, and fragment forms are reported as `unknown`, not guessed into local paths. Unknowns do not change the exit code. Symlink escapes and filesystem inspection errors are unsafe input errors and return exit `2`.

## Scope and limitations

- Supported input: SARIF `2.1.0`, local files, POSIX checkout semantics.
- Supported output: deterministic text and versioned JSON.
- Not supported in V1: Windows drive or UNC paths, remote URI schemes, scanner-specific conversion, SARIF rewriting, fingerprints, upload, or undocumented GitHub behavior.
- Location-less results are not rejected merely for lacking a location.
- Generic schema validation remains useful and complementary; this tool is not a replacement for a full SARIF validator.

## Security and privacy

The runtime is offline and has no third-party dependencies. Inputs are bounded to 16 MiB per file and 32 files, 1,024 runs, 100,000 results, and 200,000 locations per invocation. Artifact URIs are limited to 4,096 bytes. See [SECURITY.md](SECURITY.md) for the reporting policy and current security boundaries.

## Development

```sh
go test ./...
go test -race ./...
go vet ./...
gofmt -w cmd internal
scripts/test-policy.sh
scripts/test-release.sh
go build -trimpath -buildvcs=false -o /tmp/github-sarif-preflight ./cmd/github-sarif-preflight
scripts/test-performance.sh /tmp/github-sarif-preflight
scripts/test-alternatives.sh /path/to/github-sarif-preflight /path/to/sarif
```

The release test builds every supported target twice, compares bytes and checksum indexes, verifies archive contents, and executes the host binary. The performance gate runs 100,000 results with a 30-second and 256-MiB budget. The policy gate requires a stdlib-only runtime module graph, Apache-2.0 text, immutable Action pins, no runtime network/process imports, and no tracked private-key or common token patterns.

The alternative regression requires exactly Sarif.Multitool `5.5.0` and `jq`. It proves that the generic validator and JSON shape check accept the four consumer-profile fixtures while this tool returns `GSP001` through `GSP004`.

## Uninstall

Remove the `github-sarif-preflight` binary from the directory reported by `go env GOBIN` or from `$(go env GOPATH)/bin`. The tool creates no configuration, cache, account, network resource, or background service.

## License

Apache License 2.0. See [LICENSE](LICENSE).
