# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic Versioning.

## [Unreleased]

<!-- AUTO-UNRELEASED:START -->

<!-- AUTO-UNRELEASED:END -->

## [0.8.0] - 2026-07-12

### Added
- drop support for legacy twelvedata errors with 200 http status codes and real error code in response body

## [0.7.0] - 2026-06-23

### Added
- add earnings endpoint


## [0.6.0] - 2026-06-02

### Added
- add support for ETFs endpoints and update documentation

## [0.5.0] - 2026-05-01

### Added
- support revenue forecasting endpoint
- support earnings forecasting endpoint
- support consolidated income_statement endpoint

## [0.4.0] - 2026-04-25

### Added
- support market_cap endpoint

### Fixed
- include final commit in changelog generation

### Chore
- add local changelog automation and backfill v0.3.0

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

[Unreleased]: https://github.com/christiankozalla/twelvedata-go/compare/v0.7.0...HEAD
[0.6.0]: https://github.com/christiankozalla/twelvedata-go/releases/tag/v0.6.0
[0.5.0]: https://github.com/christiankozalla/twelvedata-go/releases/tag/v0.5.0
[0.4.0]: https://github.com/christiankozalla/twelvedata-go/releases/tag/v0.4.0
[0.3.0]: https://github.com/christiankozalla/twelvedata-go/releases/tag/v0.3.0
[0.2.0]: https://github.com/christiankozalla/twelvedata-go/releases/tag/v0.2.0
[0.1.0]: https://github.com/christiankozalla/twelvedata-go/releases/tag/v0.1.0
