package twelvedata

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/url"
	"strings"
)

// TimeSeriesParams enumerates filters for /time_series.
type TimeSeriesParams struct {
	Symbol        string
	Interval      string
	Exchange      string
	Country       string
	Type          string
	OutputSize    *int
	StartDate     string
	EndDate       string
	DP            *int
	Timezone      string
	Order         string
	Prepost       *bool
	Date          string
	MICCode       string
	PreviousClose *bool
	Adjust        string
}

// TimeSeriesResponse captures the raw /time_series response.
type TimeSeriesResponse struct {
	Meta   TimeSeriesMeta    `json:"meta"`
	Values []TimeSeriesValue `json:"values"`
	Status string            `json:"status,omitempty"`
}

// TimeSeriesMeta contains the metadata returned with /time_series results.
type TimeSeriesMeta struct {
	Symbol           string `json:"symbol,omitempty"`
	Interval         string `json:"interval,omitempty"`
	Currency         string `json:"currency,omitempty"`
	ExchangeTimezone string `json:"exchange_timezone,omitempty"`
	Exchange         string `json:"exchange,omitempty"`
	MicCode          string `json:"mic_code,omitempty"`
	Type             string `json:"type,omitempty"`

	// Backward compatibility for APIs that may still use `timezone` in meta.
	Timezone string `json:"timezone,omitempty"`
}

// TimeSeriesValue captures a single time series datapoint.
type TimeSeriesValue struct {
	Datetime      string `json:"datetime,omitempty"`
	Open          string `json:"open,omitempty"`
	High          string `json:"high,omitempty"`
	Low           string `json:"low,omitempty"`
	Close         string `json:"close,omitempty"`
	Volume        string `json:"volume,omitempty"`
	PreviousClose string `json:"previous_close,omitempty"`
}

// TimeSeries provides helpers for composing price and indicator queries.
type TimeSeries struct {
	client       *Client
	params       TimeSeriesParams
	priceEnabled bool
}

// TimeSeries returns a builder for /time_series queries.
func (c *Client) TimeSeries(params TimeSeriesParams) *TimeSeries {
	builder := &TimeSeries{
		client:       c,
		params:       params,
		priceEnabled: true,
	}

	if builder.params.Interval == "" {
		builder.params.Interval = "1min"
	}
	if builder.params.OutputSize == nil {
		defaultOutput := 30
		builder.params.OutputSize = &defaultOutput
	}
	if builder.params.DP == nil {
		defaultDP := 5
		builder.params.DP = &defaultDP
	}
	if builder.params.Timezone == "" {
		builder.params.Timezone = "Exchange"
	}
	if builder.params.Order == "" {
		builder.params.Order = "desc"
	}

	return builder
}

// AsJSON fetches the time series and normalizes it into a slice of datapoints.
func (ts *TimeSeries) AsJSON(ctx context.Context) ([]map[string]any, error) {
	if !ts.priceEnabled {
		return nil, fmt.Errorf("twelvedata: price endpoint disabled")
	}

	raw, err := ts.priceRequest().AsNormalizedJSON(ctx)
	if err != nil {
		return nil, err
	}

	series, err := toMapSlice(raw)
	if err != nil {
		return nil, err
	}
	return series, nil
}

// AsResponse fetches the full /time_series payload into typed metadata and values.
func (ts *TimeSeries) AsResponse(ctx context.Context) (*TimeSeriesResponse, error) {
	if !ts.priceEnabled {
		return nil, fmt.Errorf("twelvedata: price endpoint disabled")
	}

	var response TimeSeriesResponse
	if err := ts.priceRequest().AsJSON(ctx, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// AsCSV fetches the time series as CSV rows.
func (ts *TimeSeries) AsCSV(ctx context.Context) ([][]string, error) {
	if !ts.priceEnabled {
		return nil, fmt.Errorf("twelvedata: price endpoint disabled")
	}

	payload, err := ts.priceRequest().AsCSV(ctx)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(strings.NewReader(payload))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	return records, nil
}

// AsURL builds the URLs required to retrieve the configured series.
func (ts *TimeSeries) AsURL() ([]string, error) {
	if !ts.priceEnabled {
		return nil, fmt.Errorf("twelvedata: price endpoint disabled")
	}

	url, err := ts.priceRequest().AsURL()
	if err != nil {
		return nil, err
	}
	return []string{url}, nil
}

func (ts *TimeSeries) priceRequest() *Request {
	values := url.Values{}
	addString(values, "symbol", ts.params.Symbol)
	addString(values, "interval", ts.params.Interval)
	addString(values, "exchange", ts.params.Exchange)
	addString(values, "country", ts.params.Country)
	addString(values, "type", ts.params.Type)
	addInt(values, "outputsize", ts.params.OutputSize)
	addString(values, "start_date", ts.params.StartDate)
	addString(values, "end_date", ts.params.EndDate)
	addInt(values, "dp", ts.params.DP)
	addString(values, "timezone", ts.params.Timezone)
	addString(values, "order", ts.params.Order)
	addBool(values, "prepost", ts.params.Prepost)
	addString(values, "date", ts.params.Date)
	addString(values, "mic_code", ts.params.MICCode)
	addBool(values, "previous_close", ts.params.PreviousClose)
	addString(values, "adjust", ts.params.Adjust)
	return ts.client.newRequest("/time_series", values)
}

func toMapSlice(raw interface{}) ([]map[string]any, error) {
	list, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("twelvedata: unexpected time series payload %T", raw)
	}

	result := make([]map[string]any, 0, len(list))
	for _, item := range list {
		obj, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("twelvedata: unexpected datapoint type %T", item)
		}
		result = append(result, obj)
	}
	return result, nil
}
