# Contributing

Thank you for helping improve `github-sarif-preflight`.

## Before opening a change

- Keep the scope limited to documented GitHub Code Scanning SARIF behavior.
- Add a synthetic, license-safe fixture for every behavior change.
- Never include private SARIF, source code, credentials, personal data, or proprietary scanner output.
- Do not add network access, telemetry, upload, or automatic rewriting.

## Development checks

Use Go 1.23 or later and run:

```sh
gofmt -w cmd internal
go test ./...
go vet ./...
```

Tests must cover the CLI exit code and both text and JSON contracts when diagnostics change. New diagnostic IDs or support-matrix expansions require reproducible external user evidence, not only a hypothetical fixture.

## Reporting security issues

Follow [SECURITY.md](SECURITY.md). Do not include secrets or sensitive SARIF in a public issue.
