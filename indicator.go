package twelvedata

import (
	"net/url"
	"strings"
)

// IndicatorParams captures common query parameters for technical indicator endpoints.
type IndicatorParams struct {
	Symbol     string
	Interval   string
	Exchange   string
	Country    string
	Type       string
	SeriesType string
	TimePeriod *int
	OutputSize *int
	StartDate  string
	EndDate    string
	DP         *int
	Timezone   string
	Order      string
	Prepost    *bool
	MICCode    string
	Extra      map[string]string
}

// Indicator returns a request configured for the given indicator endpoint (e.g. rsi, adx).
func (c *Client) Indicator(name string, params IndicatorParams) *Request {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	if trimmed == "" {
		trimmed = "indicator"
	}
	path := "/" + strings.TrimLeft(trimmed, "/")

	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "interval", params.Interval)
	addString(values, "exchange", params.Exchange)
	addString(values, "country", params.Country)
	addString(values, "type", params.Type)
	addString(values, "series_type", params.SeriesType)
	addInt(values, "time_period", params.TimePeriod)
	addInt(values, "outputsize", params.OutputSize)
	addString(values, "start_date", params.StartDate)
	addString(values, "end_date", params.EndDate)
	addInt(values, "dp", params.DP)
	addString(values, "timezone", params.Timezone)
	addString(values, "order", params.Order)
	addBool(values, "prepost", params.Prepost)
	addString(values, "mic_code", params.MICCode)

	for key, value := range params.Extra {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		values.Set(key, value)
	}

	return c.newRequest(path, values)
}

// WILLRResponse captures the full /willr payload.
type WILLRResponse struct {
	Meta   WILLRMeta    `json:"meta"`
	Values []WILLRValue `json:"values"`
	Status string       `json:"status,omitempty"`
}

// WILLRMeta captures metadata returned by /willr.
type WILLRMeta struct {
	Symbol           string             `json:"symbol,omitempty"`
	Interval         string             `json:"interval,omitempty"`
	Currency         string             `json:"currency,omitempty"`
	ExchangeTimezone string             `json:"exchange_timezone,omitempty"`
	Exchange         string             `json:"exchange,omitempty"`
	MicCode          string             `json:"mic_code,omitempty"`
	Type             string             `json:"type,omitempty"`
	Indicator        WILLRIndicatorMeta `json:"indicator,omitempty"`
}

// WILLRIndicatorMeta captures indicator-specific metadata for /willr.
type WILLRIndicatorMeta struct {
	Name       string `json:"name,omitempty"`
	TimePeriod int    `json:"time_period,omitempty"`
}

// WILLRValue captures a single Williams %R datapoint.
type WILLRValue struct {
	Datetime      string `json:"datetime,omitempty"`
	WILLR         string `json:"willr,omitempty"`
	Open          string `json:"open,omitempty"`
	High          string `json:"high,omitempty"`
	Low           string `json:"low,omitempty"`
	Close         string `json:"close,omitempty"`
	PreviousClose string `json:"previous_close,omitempty"`
}

// ADXResponse captures the full /adx payload.
type ADXResponse struct {
	Meta   ADXMeta    `json:"meta"`
	Values []ADXValue `json:"values"`
	Status string     `json:"status,omitempty"`
}

// ADXMeta captures metadata returned by /adx.
type ADXMeta struct {
	Symbol           string           `json:"symbol,omitempty"`
	Interval         string           `json:"interval,omitempty"`
	Currency         string           `json:"currency,omitempty"`
	ExchangeTimezone string           `json:"exchange_timezone,omitempty"`
	Exchange         string           `json:"exchange,omitempty"`
	MicCode          string           `json:"mic_code,omitempty"`
	Type             string           `json:"type,omitempty"`
	Indicator        ADXIndicatorMeta `json:"indicator,omitempty"`
}

// ADXIndicatorMeta captures indicator-specific metadata for /adx.
type ADXIndicatorMeta struct {
	Name       string `json:"name,omitempty"`
	TimePeriod int    `json:"time_period,omitempty"`
}

// ADXValue captures a single ADX datapoint.
type ADXValue struct {
	Datetime      string `json:"datetime,omitempty"`
	ADX           string `json:"adx,omitempty"`
	Open          string `json:"open,omitempty"`
	High          string `json:"high,omitempty"`
	Low           string `json:"low,omitempty"`
	Close         string `json:"close,omitempty"`
	PreviousClose string `json:"previous_close,omitempty"`
}

// PlusDIResponse captures the full /plus_di payload.
type PlusDIResponse struct {
	Meta   PlusDIMeta    `json:"meta"`
	Values []PlusDIValue `json:"values"`
	Status string        `json:"status,omitempty"`
}

// PlusDIMeta captures metadata returned by /plus_di.
type PlusDIMeta struct {
	Symbol           string              `json:"symbol,omitempty"`
	Interval         string              `json:"interval,omitempty"`
	Currency         string              `json:"currency,omitempty"`
	ExchangeTimezone string              `json:"exchange_timezone,omitempty"`
	Exchange         string              `json:"exchange,omitempty"`
	MicCode          string              `json:"mic_code,omitempty"`
	Type             string              `json:"type,omitempty"`
	Indicator        PlusDIIndicatorMeta `json:"indicator,omitempty"`
}

// PlusDIIndicatorMeta captures indicator-specific metadata for /plus_di.
type PlusDIIndicatorMeta struct {
	Name       string `json:"name,omitempty"`
	TimePeriod int    `json:"time_period,omitempty"`
}

// PlusDIValue captures a single Plus DI datapoint.
type PlusDIValue struct {
	Datetime      string `json:"datetime,omitempty"`
	PlusDI        string `json:"plus_di,omitempty"`
	Open          string `json:"open,omitempty"`
	High          string `json:"high,omitempty"`
	Low           string `json:"low,omitempty"`
	Close         string `json:"close,omitempty"`
	PreviousClose string `json:"previous_close,omitempty"`
}

// MinusDIResponse captures the full /minus_di payload.
type MinusDIResponse struct {
	Meta   MinusDIMeta    `json:"meta"`
	Values []MinusDIValue `json:"values"`
	Status string         `json:"status,omitempty"`
}

// MinusDIMeta captures metadata returned by /minus_di.
type MinusDIMeta struct {
	Symbol           string               `json:"symbol,omitempty"`
	Interval         string               `json:"interval,omitempty"`
	Currency         string               `json:"currency,omitempty"`
	ExchangeTimezone string               `json:"exchange_timezone,omitempty"`
	Exchange         string               `json:"exchange,omitempty"`
	MicCode          string               `json:"mic_code,omitempty"`
	Type             string               `json:"type,omitempty"`
	Indicator        MinusDIIndicatorMeta `json:"indicator,omitempty"`
}

// MinusDIIndicatorMeta captures indicator-specific metadata for /minus_di.
type MinusDIIndicatorMeta struct {
	Name       string `json:"name,omitempty"`
	TimePeriod int    `json:"time_period,omitempty"`
}

// MinusDIValue captures a single Minus DI datapoint.
type MinusDIValue struct {
	Datetime      string `json:"datetime,omitempty"`
	MinusDI       string `json:"minus_di,omitempty"`
	Open          string `json:"open,omitempty"`
	High          string `json:"high,omitempty"`
	Low           string `json:"low,omitempty"`
	Close         string `json:"close,omitempty"`
	PreviousClose string `json:"previous_close,omitempty"`
}

// WILLRParams captures query parameters for the /willr endpoint.
//
// Python-parity fields (symbol, interval, exchange, country, type, time_period,
// outputsize, start_date, end_date, dp, timezone, order, prepost, mic_code)
// are included directly, and additional optional identifier/series fields from
// current Twelve Data docs are also supported.
type WILLRParams struct {
	Symbol        string
	FIGI          string
	ISIN          string
	CUSIP         string
	Interval      string
	Exchange      string
	MICCode       string
	Country       string
	Type          string
	TimePeriod    *int
	OutputSize    *int
	DP            *int
	Order         string
	Timezone      string
	Date          string
	StartDate     string
	EndDate       string
	Prepost       *bool
	IncludeOHLC   *bool
	PreviousClose *bool
	Adjust        string
}

// WILLR returns a request configured for the Williams %R momentum indicator endpoint.
func (c *Client) WILLR(params WILLRParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "figi", params.FIGI)
	addString(values, "isin", params.ISIN)
	addString(values, "cusip", params.CUSIP)
	addString(values, "interval", params.Interval)
	addString(values, "exchange", params.Exchange)
	addString(values, "mic_code", params.MICCode)
	addString(values, "country", params.Country)
	addString(values, "type", params.Type)
	addInt(values, "time_period", params.TimePeriod)
	addInt(values, "outputsize", params.OutputSize)
	addInt(values, "dp", params.DP)
	addString(values, "order", params.Order)
	addString(values, "timezone", params.Timezone)
	addString(values, "date", params.Date)
	addString(values, "start_date", params.StartDate)
	addString(values, "end_date", params.EndDate)
	addBool(values, "prepost", params.Prepost)
	addBool(values, "include_ohlc", params.IncludeOHLC)
	addBool(values, "previous_close", params.PreviousClose)
	addString(values, "adjust", params.Adjust)

	return c.newRequest("/willr", values)
}

// ADXParams captures query parameters for the /adx endpoint.
//
// Python-parity fields (symbol, interval, exchange, country, type, time_period,
// outputsize, start_date, end_date, dp, timezone, order, prepost, mic_code)
// are included directly, and additional optional identifier/series fields from
// current Twelve Data docs are also supported.
type ADXParams struct {
	Symbol        string
	FIGI          string
	ISIN          string
	CUSIP         string
	Interval      string
	Exchange      string
	MICCode       string
	Country       string
	Type          string
	TimePeriod    *int
	OutputSize    *int
	DP            *int
	Order         string
	Timezone      string
	Date          string
	StartDate     string
	EndDate       string
	Prepost       *bool
	IncludeOHLC   *bool
	PreviousClose *bool
	Adjust        string
}

// ADX returns a request configured for the Average Directional Index endpoint.
func (c *Client) ADX(params ADXParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "figi", params.FIGI)
	addString(values, "isin", params.ISIN)
	addString(values, "cusip", params.CUSIP)
	addString(values, "interval", params.Interval)
	addString(values, "exchange", params.Exchange)
	addString(values, "mic_code", params.MICCode)
	addString(values, "country", params.Country)
	addString(values, "type", params.Type)
	addInt(values, "time_period", params.TimePeriod)
	addInt(values, "outputsize", params.OutputSize)
	addInt(values, "dp", params.DP)
	addString(values, "order", params.Order)
	addString(values, "timezone", params.Timezone)
	addString(values, "date", params.Date)
	addString(values, "start_date", params.StartDate)
	addString(values, "end_date", params.EndDate)
	addBool(values, "prepost", params.Prepost)
	addBool(values, "include_ohlc", params.IncludeOHLC)
	addBool(values, "previous_close", params.PreviousClose)
	addString(values, "adjust", params.Adjust)

	return c.newRequest("/adx", values)
}

// PlusDIParams captures query parameters for the /plus_di endpoint.
type PlusDIParams struct {
	Symbol        string
	FIGI          string
	ISIN          string
	CUSIP         string
	Interval      string
	Exchange      string
	MICCode       string
	Country       string
	Type          string
	TimePeriod    *int
	OutputSize    *int
	DP            *int
	Order         string
	Timezone      string
	Date          string
	StartDate     string
	EndDate       string
	Prepost       *bool
	IncludeOHLC   *bool
	PreviousClose *bool
	Adjust        string
}

// PlusDI returns a request configured for the Plus Directional Indicator endpoint.
func (c *Client) PlusDI(params PlusDIParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "figi", params.FIGI)
	addString(values, "isin", params.ISIN)
	addString(values, "cusip", params.CUSIP)
	addString(values, "interval", params.Interval)
	addString(values, "exchange", params.Exchange)
	addString(values, "mic_code", params.MICCode)
	addString(values, "country", params.Country)
	addString(values, "type", params.Type)
	addInt(values, "time_period", params.TimePeriod)
	addInt(values, "outputsize", params.OutputSize)
	addInt(values, "dp", params.DP)
	addString(values, "order", params.Order)
	addString(values, "timezone", params.Timezone)
	addString(values, "date", params.Date)
	addString(values, "start_date", params.StartDate)
	addString(values, "end_date", params.EndDate)
	addBool(values, "prepost", params.Prepost)
	addBool(values, "include_ohlc", params.IncludeOHLC)
	addBool(values, "previous_close", params.PreviousClose)
	addString(values, "adjust", params.Adjust)

	return c.newRequest("/plus_di", values)
}

// MinusDIParams captures query parameters for the /minus_di endpoint.
type MinusDIParams struct {
	Symbol        string
	FIGI          string
	ISIN          string
	CUSIP         string
	Interval      string
	Exchange      string
	MICCode       string
	Country       string
	Type          string
	TimePeriod    *int
	OutputSize    *int
	DP            *int
	Order         string
	Timezone      string
	Date          string
	StartDate     string
	EndDate       string
	Prepost       *bool
	IncludeOHLC   *bool
	PreviousClose *bool
	Adjust        string
}

// MinusDI returns a request configured for the Minus Directional Indicator endpoint.
func (c *Client) MinusDI(params MinusDIParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "figi", params.FIGI)
	addString(values, "isin", params.ISIN)
	addString(values, "cusip", params.CUSIP)
	addString(values, "interval", params.Interval)
	addString(values, "exchange", params.Exchange)
	addString(values, "mic_code", params.MICCode)
	addString(values, "country", params.Country)
	addString(values, "type", params.Type)
	addInt(values, "time_period", params.TimePeriod)
	addInt(values, "outputsize", params.OutputSize)
	addInt(values, "dp", params.DP)
	addString(values, "order", params.Order)
	addString(values, "timezone", params.Timezone)
	addString(values, "date", params.Date)
	addString(values, "start_date", params.StartDate)
	addString(values, "end_date", params.EndDate)
	addBool(values, "prepost", params.Prepost)
	addBool(values, "include_ohlc", params.IncludeOHLC)
	addBool(values, "previous_close", params.PreviousClose)
	addString(values, "adjust", params.Adjust)

	return c.newRequest("/minus_di", values)
}
