// Package twelvedata provides a Go client for the Twelve Data REST API and
// realtime price WebSocket.
//
// The current Go port already covers a practical base subset:
//   - instrument catalogs such as stocks, forex pairs, exchanges, ETFs, ETF directories/families/types, funds, and bonds
//   - market data endpoints such as exchange rate, currency conversion, quote, price, eod, logo, profile, market cap, statistics, earnings, earnings estimate, revenue estimate, income statement, income statement consolidated, and last changes
//   - options endpoints for expiration and chain
//   - time series and technical indicators
//   - realtime price events and subscription status over a typed WebSocket stream
//
// The package is designed so new GET endpoints can be added with minimal work:
//  1. define a params struct for the endpoint
//  2. translate params into url.Values with the shared addString/addBool/addInt/addFloat helpers
//  3. return c.newRequest("/endpoint_name", values)
//  4. add any higher-level response helper only when the endpoint needs custom shaping, as time series does
//
// Response handling is centralized in Request:
//   - AsURL builds a fully qualified request URL
//   - AsRawJSON returns the raw JSON payload
//   - AsJSON decodes into a caller-provided response type
//   - AsNormalized decodes the normalized payload into a caller-provided response type
//   - AsNormalizedJSON mirrors the Python client's common normalization rules
//   - AsCSV returns CSV payloads as text
//
// Twelve Data reports API errors through HTTP error status codes. The shared transport converts
// those responses into APIError so future endpoints inherit the same behavior automatically.
package twelvedata
