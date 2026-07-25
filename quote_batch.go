package twelvedata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const minimumQuoteBatchSize = 2

// QuoteBatchParams enumerates filters for a multi-symbol /quote request.
// MICCode applies to every symbol in the batch.
type QuoteBatchParams struct {
	Symbols          []string
	Interval         string
	Exchange         string
	Country          string
	VolumeTimePeriod string
	Type             string
	DP               *int
	Timezone         string
	Prepost          *bool
	MICCode          string
	EOD              string
	RollingPeriod    string
}

// QuoteBatchResult contains exactly one quote or per-symbol error.
type QuoteBatchResult struct {
	Quote *QuoteResponse
	Error *APIError
}

// QuoteBatchResponse preserves the provider's symbol keys.
type QuoteBatchResponse struct {
	Items map[string]QuoteBatchResult
}

// QuoteBatchRequest is a validated multi-symbol /quote request.
type QuoteBatchRequest struct {
	request *Request
	symbols []string
}

// QuoteBatch creates one /quote request for two or more distinct symbols.
func (c *Client) QuoteBatch(params QuoteBatchParams) (*QuoteBatchRequest, error) {
	symbols, err := normalizeQuoteBatchSymbols(params.Symbols)
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	addString(values, "symbol", strings.Join(symbols, ","))
	addString(values, "interval", params.Interval)
	addString(values, "exchange", params.Exchange)
	addString(values, "country", params.Country)
	addString(values, "volume_time_period", params.VolumeTimePeriod)
	addString(values, "type", params.Type)
	addInt(values, "dp", params.DP)
	addString(values, "timezone", params.Timezone)
	addBool(values, "prepost", params.Prepost)
	addString(values, "mic_code", params.MICCode)
	addString(values, "eod", params.EOD)
	addString(values, "rolling_period", params.RollingPeriod)
	return &QuoteBatchRequest{
		request: c.newRequest("/quote", values),
		symbols: symbols,
	}, nil
}

// AsJSON executes and decodes a keyed batch response. A top-level provider
// failure is returned as an error; member failures remain isolated in Items.
func (r *QuoteBatchRequest) AsJSON(ctx context.Context) (*QuoteBatchResponse, error) {
	if r == nil || r.request == nil {
		return nil, fmt.Errorf("twelvedata: quote batch request is not initialized")
	}
	payload, err := r.request.AsRawJSON(ctx)
	if err != nil {
		return nil, err
	}
	return decodeQuoteBatchResponse(payload, r.symbols)
}

func normalizeQuoteBatchSymbols(values []string) ([]string, error) {
	symbols := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		symbol := strings.TrimSpace(value)
		if symbol == "" {
			return nil, fmt.Errorf("twelvedata: quote batch symbol must not be empty")
		}
		if strings.Contains(symbol, ",") {
			return nil, fmt.Errorf("twelvedata: quote batch symbol %q must not contain a comma", symbol)
		}
		key := strings.ToUpper(symbol)
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("twelvedata: duplicate quote batch symbol %q", symbol)
		}
		seen[key] = struct{}{}
		symbols = append(symbols, symbol)
	}
	if len(symbols) < minimumQuoteBatchSize {
		return nil, fmt.Errorf("twelvedata: quote batch requires at least %d distinct symbols", minimumQuoteBatchSize)
	}
	return symbols, nil
}

func decodeQuoteBatchResponse(
	payload []byte,
	requestedSymbols []string,
) (*QuoteBatchResponse, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}

	if topLevelError(raw) {
		var item apiErrorPayload
		_ = json.Unmarshal(payload, &item)
		return nil, &APIError{
			Code:       item.Code,
			HTTPStatus: item.Code,
			Message:    item.Message,
		}
	}
	if data, exists := raw["data"]; exists {
		var wrapped map[string]json.RawMessage
		if err := json.Unmarshal(data, &wrapped); err != nil {
			return nil, fmt.Errorf("twelvedata: decode quote batch data: %w", err)
		}
		raw = wrapped
	}

	response := &QuoteBatchResponse{Items: make(map[string]QuoteBatchResult, len(requestedSymbols))}
	requestedByKey := make(map[string]string, len(requestedSymbols))
	for _, symbol := range requestedSymbols {
		requestedByKey[strings.ToUpper(strings.TrimSpace(symbol))] = symbol
	}

	if encoded, single := singleQuoteBatchObject(raw); single {
		var quote QuoteResponse
		if err := json.Unmarshal(encoded, &quote); err != nil {
			return nil, fmt.Errorf("twelvedata: decode single quote batch response: %w", err)
		}
		requestedSymbol, requested := requestedByKey[strings.ToUpper(strings.TrimSpace(quote.Symbol))]
		if requested {
			response.Items[requestedSymbol] = quoteBatchQuoteResult(quote)
		}
	} else {
		for providerKey, encoded := range raw {
			requestedSymbol, requested := requestedByKey[strings.ToUpper(strings.TrimSpace(providerKey))]
			if !requested {
				continue
			}
			var itemError struct {
				Code    int    `json:"code"`
				Status  string `json:"status"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(encoded, &itemError); err != nil {
				response.Items[requestedSymbol] = QuoteBatchResult{
					Error: &APIError{Message: "invalid quote batch item"},
				}
				continue
			}
			if strings.EqualFold(itemError.Status, "error") || itemError.Code != 0 {
				response.Items[requestedSymbol] = QuoteBatchResult{Error: &APIError{
					Code:       itemError.Code,
					HTTPStatus: itemError.Code,
					Message:    itemError.Message,
				}}
				continue
			}

			var quote QuoteResponse
			if err := json.Unmarshal(encoded, &quote); err != nil {
				response.Items[requestedSymbol] = QuoteBatchResult{
					Error: &APIError{Message: "invalid quote batch item"},
				}
				continue
			}
			response.Items[requestedSymbol] = quoteBatchQuoteResult(quote)
		}
	}

	for _, symbol := range requestedSymbols {
		if _, exists := response.Items[symbol]; !exists {
			response.Items[symbol] = QuoteBatchResult{Error: &APIError{
				Message: "quote missing from batch response",
			}}
		}
	}
	return response, nil
}

func quoteBatchQuoteResult(quote QuoteResponse) QuoteBatchResult {
	if strings.TrimSpace(quote.Symbol) == "" {
		return QuoteBatchResult{Error: &APIError{Message: "empty quote batch item"}}
	}
	return QuoteBatchResult{Quote: &quote}
}

func singleQuoteBatchObject(raw map[string]json.RawMessage) (json.RawMessage, bool) {
	if _, exists := raw["symbol"]; !exists {
		return nil, false
	}
	encoded, err := json.Marshal(raw)
	return encoded, err == nil
}

func topLevelError(raw map[string]json.RawMessage) bool {
	encoded, exists := raw["status"]
	if !exists {
		return false
	}
	var status string
	return json.Unmarshal(encoded, &status) == nil && strings.EqualFold(status, "error")
}
