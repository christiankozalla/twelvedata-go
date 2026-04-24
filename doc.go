// Package twelvedata provides a read-only Go client for the Twelve Data REST API.
//
// The current Go port already covers a practical base subset:
//   - instrument catalogs such as stocks, forex pairs, exchanges, ETFs, funds, and bonds
//   - market data endpoints such as exchange rate, currency conversion, quote, price, eod, logo, and statistics
//   - options endpoints for expiration and chain
//   - time series and technical indicators
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
// Twelve Data can return logical API errors in JSON bodies even when the HTTP status is 200.
// The shared transport detects those responses and returns APIError so future endpoints inherit
// the same behavior automatically.
package twelvedata
