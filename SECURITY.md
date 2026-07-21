# Security Policy

## Supported versions

No public release exists yet. Security fixes are applied to the default branch during development. A supported-version table will be added with the first release.

## Reporting a vulnerability

Use GitHub private vulnerability reporting when it becomes available for the public repository. Until then, do not post secrets, private SARIF, source code, credentials, or exploit details in a public issue. A safe public issue may state only that a private reporting path is needed.

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
