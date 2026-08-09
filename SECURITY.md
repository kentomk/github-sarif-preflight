# Security Policy

## Supported versions

The published `v0.1.3` release is supported. Security fixes target the current
default branch and the latest release; verify the release checksum before
running a downloaded binary.

## Reporting a vulnerability

Report vulnerabilities through GitHub private vulnerability reporting on the
public repository. Do not post secrets, private SARIF, source code,
credentials, or exploit details in a public issue. If private reporting is
unavailable, provide only a minimal synthetic reproducer with neutral data.

## Security boundaries

- The CLI is offline, read-only, and has no telemetry or runtime network client.
- It does not upload or rewrite SARIF and does not read source-file contents.
- Diagnostics do not echo SARIF messages or source snippets.
- Each input is valid UTF-8 and at most 16 MiB; each invocation accepts at most 32 files, 1,024 runs, 100,000 results, and 200,000 locations.
- Artifact URIs, base IDs, and rule IDs are bounded before they can become diagnostics.
- Repository roots are canonicalized, lexical escapes are rejected before filesystem inspection, and symlink escapes return an unsafe-input error.
- The composite Action executes the same CLI contract, requires no token, and either builds the selected immutable revision or executes an explicitly supplied binary path.
- Run the development build only against a stable checkout that you control.

Security reports about undocumented GitHub behavior must include a public specification or a minimal synthetic reproduction before that behavior can become an error diagnostic.
