# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic Versioning.

## [Unreleased]

## [0.2.0] - 2026-04-24

### Added
- `Logo` endpoint support with `LogoParams` and `LogoResponse`.

### Changed
- README examples and endpoint listings were streamlined and updated.
- Release flow moved to local tag-based publishing (release-please automation removed).

## [0.1.0] - 2026-04-14

### Added
- Initial standalone Go module import.
- Core read-only client transport, request helpers, and endpoint coverage baseline.
- Typed response structs for key market and indicator endpoints.
- `cmd/tdcli` helper CLI for manual endpoint calls.
- CI workflow that runs `go test ./...` on pushes and pull requests.

[Unreleased]: https://github.com/christiankozalla/twelvedata-go/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/christiankozalla/twelvedata-go/releases/tag/v0.2.0
[0.1.0]: https://github.com/christiankozalla/twelvedata-go/releases/tag/v0.1.0
