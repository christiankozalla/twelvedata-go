# Test Conversion Strategy

This document captures the initial plan for porting the Python test suite in `tests/test_client.py` to Go. The goals are to preserve functional coverage while embracing idiomatic Go testing practices.

## Guiding principles

1. **Table-driven cases** – consolidate repetitive endpoint assertions into shared test tables executed by `t.Run` subtests.
2. **Deterministic HTTP interactions** – substitute real API calls with a lightweight test server so the suite executes without external dependencies or rate limits.
3. **Thin wrappers only** – keep helper builders small to reduce the impedance when the Go client implementation lands.
4. **Progressive parity** – port tests in logical batches (instrument lists, single-resource queries, time series helpers) and expand coverage alongside the Go feature work.

## Near-term milestones

- Reproduce the “list” endpoint coverage (`get_stocks_list`, `get_funds_list`, etc.) in `client_test.go` using table-driven cases.
- Provide placeholder helpers for constructing a demo client instance, mirroring `_init_client` and `_fake_json_resp` behaviours.
- Document any Python-specific behaviours that need equivalent handling in Go (e.g. caching shortcuts) so the future implementation can satisfy them.

## Deliverables for the first batch

- `client_test.go` containing the table-driven list tests and HTTP mock scaffolding.
- Minimal supporting structures (e.g. `mockServer` helpers) under `twelvedata-go/internal/testutil` if reuse becomes helpful.
- Tracking TODO comments for pending test groups (time series, statistics, websocket coverage) to ensure visibility of remaining parity work.
