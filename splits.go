package twelvedata

import "net/url"

// SplitsParams enumerates filters for /splits.
// At least one of Symbol, FIGI, ISIN, or CUSIP is expected by the API.
type SplitsParams struct {
	Symbol    string
	FIGI      string
	ISIN      string
	CUSIP     string
	Exchange  string
	MICCode   string
	Country   string
	Range     string
	StartDate string
	EndDate   string
}

// SplitsResponse captures the /splits response.
type SplitsResponse struct {
	Meta   SplitsMeta   `json:"meta"`
	Splits []SplitValue `json:"splits"`
}

// SplitsMeta contains general instrument metadata for /splits.
type SplitsMeta struct {
	Symbol           string `json:"symbol,omitempty"`
	Name             string `json:"name,omitempty"`
	Currency         string `json:"currency,omitempty"`
	Exchange         string `json:"exchange,omitempty"`
	MICCode          string `json:"mic_code,omitempty"`
	ExchangeTimezone string `json:"exchange_timezone,omitempty"`
}

// SplitValue captures one split event. Prices dated before Date are
// multiplied by Ratio to be comparable with prices from Date onwards.
type SplitValue struct {
	Date        string  `json:"date,omitempty"`
	Description string  `json:"description,omitempty"`
	Ratio       float64 `json:"ratio,omitempty"`
	FromFactor  float64 `json:"from_factor,omitempty"`
	ToFactor    float64 `json:"to_factor,omitempty"`
}

// Splits returns the /splits resource.
func (c *Client) Splits(params SplitsParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "figi", params.FIGI)
	addString(values, "isin", params.ISIN)
	addString(values, "cusip", params.CUSIP)
	addString(values, "exchange", params.Exchange)
	addString(values, "mic_code", params.MICCode)
	addString(values, "country", params.Country)
	addString(values, "range", params.Range)
	addString(values, "start_date", params.StartDate)
	addString(values, "end_date", params.EndDate)
	return c.newRequest("/splits", values)
}
