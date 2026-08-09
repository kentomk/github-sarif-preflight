# Changelog

- Align the copy-ready immutable Action example and publisher regression with public main `4dc5c1ae`.

All notable changes to this project will be documented here.

The format is based on Keep a Changelog, and the project uses Semantic Versioning.

## [Unreleased]

### Fixed

- Refresh the immutable Action example to the verified current public main `3c654270` and reject the superseded `cb0f9f3` pin in the publisher contract.
- Create the user binary directory before the copy-ready archive install command runs in a fresh home directory.
- Refresh the copy-ready composite Action to the current successful public-main revision and reject the superseded revision in the publisher contract.
- Refresh the immutable Action example to public main `cb0f9f3` and reject the prior `29200fc` pin in the publisher contract.
- Align the copy-ready Action pin with public main 29200fc and reject the superseded 3dffb557 revision.
- Pin the copy-ready composite Action example to the current successful public main and reject the superseded revision in the publisher contract.
- Correct the archive install example to use the extracted versioned directory
  and align the immutable Action example with the current public main SHA.
- Align source-install, reproducible-package, and immutable Action examples
  with the published `v0.1.2` release and current successful public main.
- Make top-level and `check` help available on stdout with exit `0`, including
  the stable job, options, diagnostic range, and exit-code contract.
- Preserve the 30-second and 256 MiB performance gate on publisher hosts that
  do not install GNU `/usr/bin/time`, using a standard-library process fallback.

## [0.1.1] - 2026-07-22

### Changed

- Move CI, release packaging, source-build documentation, and the publisher gate from Go 1.23.12 to checksum-pinned Go 1.26.5.

## [0.1.0] - 2026-07-21

### Fixed

- Add an owner-repairable release workflow that uploads the four reproducible archives and `SHA256SUMS`.
- Align installation and Action examples with the public `v0.1.0` source release and its successful immutable main revision.

### Added

- Offline Go CLI with deterministic text and versioned JSON output.
- `GSP001` for missing inline result messages.
- `GSP002` for empty artifact URIs.
- `GSP003` for unsupported GitHub source-root base IDs.
- `GSP004` for normalized paths that escape the repository root.
- `GSP005` for missing or non-regular checkout paths.
- Percent-decoded POSIX path normalization, symlink confinement, and explicit unknown URI classification.
- UTF-8, file-size, run, result, location, URI, base ID, and rule ID resource bounds.
- Deterministic multi-run text and JSON golden contracts with explicit location index zero.
- Composite GitHub Action with safe, diagnostic, and invalid-input CI smoke coverage.
- Pinned Sarif.Multitool 5.5.0 and `jq` false-negative regression against `GSP001` through `GSP004` fixtures.
- Reproducible Linux/macOS release archives for `amd64` and `arm64` with a SHA-256 index and embedded version.
- Race, stdlib-only license/secret/static policy, and 100,000-result performance/memory release gates.
- Synthetic safe, failure, and invalid-input fixtures with automated tests.
- English-first installation, 60-second quickstart, security, and contribution documentation.
