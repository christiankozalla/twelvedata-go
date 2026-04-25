# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic Versioning.

## [Unreleased]

<!-- AUTO-UNRELEASED:START -->
_No unreleased changes._
<!-- AUTO-UNRELEASED:END -->

## [0.3.0] - 2026-04-24

### Added
- `Profile` endpoint support with `ProfileParams` and `ProfileResponse`.
- `LastChanges` endpoint support with `LastChangesParams` and `LastChangesResponse`.
- `IncomeStatement` endpoint support with `IncomeStatementParams` and `IncomeStatementResponse`.
- `Statistics` endpoint support with `StatisticsParams` and `StatisticsResponse`.

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

[Unreleased]: https://github.com/christiankozalla/twelvedata-go/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/christiankozalla/twelvedata-go/releases/tag/v0.3.0
[0.2.0]: https://github.com/christiankozalla/twelvedata-go/releases/tag/v0.2.0
[0.1.0]: https://github.com/christiankozalla/twelvedata-go/releases/tag/v0.1.0
