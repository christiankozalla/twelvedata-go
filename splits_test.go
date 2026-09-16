package twelvedata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSplitsTypedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"meta": {
				"symbol": "BRCC",
				"name": "BRC Inc.",
				"currency": "USD",
				"exchange": "NYSE",
				"mic_code": "XNYS",
				"exchange_timezone": "America/New_York"
			},
			"splits": [
				{"date": "2026-08-24", "description": "1-for-10 split", "ratio": 10, "from_factor": 1, "to_factor": 10},
				{"date": "2020-08-31", "description": "4-for-1 split", "ratio": 0.25, "from_factor": 4, "to_factor": 1}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient("demo", WithBaseURL(server.URL))
	var response SplitsResponse
	err := client.Splits(SplitsParams{Symbol: "BRCC", MICCode: "XNYS"}).AsJSON(context.Background(), &response)
	if err != nil {
		t.Fatalf("AsJSON: %v", err)
	}
	if response.Meta.Symbol != "BRCC" {
		t.Fatalf("expected symbol BRCC, got %q", response.Meta.Symbol)
	}
	if response.Meta.MICCode != "XNYS" {
		t.Fatalf("expected mic_code XNYS, got %q", response.Meta.MICCode)
	}
	if response.Meta.ExchangeTimezone != "America/New_York" {
		t.Fatalf("expected exchange_timezone America/New_York, got %q", response.Meta.ExchangeTimezone)
	}
	if len(response.Splits) != 2 {
		t.Fatalf("expected 2 splits, got %d", len(response.Splits))
	}

	reverse := response.Splits[0]
	if reverse.Date != "2026-08-24" || reverse.Description != "1-for-10 split" {
		t.Fatalf("unexpected reverse split %+v", reverse)
	}
	if reverse.Ratio != 10 || reverse.FromFactor != 1 || reverse.ToFactor != 10 {
		t.Fatalf("unexpected reverse split factors %+v", reverse)
	}

	forward := response.Splits[1]
	if forward.Ratio != 0.25 || forward.FromFactor != 4 || forward.ToFactor != 1 {
		t.Fatalf("unexpected forward split factors %+v", forward)
	}
}

func TestSplitsEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta": {"symbol": "AAPL", "mic_code": "XNGS"}, "splits": []}`))
	}))
	defer server.Close()

	client := NewClient("demo", WithBaseURL(server.URL))
	var response SplitsResponse
	if err := client.Splits(SplitsParams{Symbol: "AAPL", Range: "1m"}).AsJSON(context.Background(), &response); err != nil {
		t.Fatalf("AsJSON: %v", err)
	}
	if response.Splits == nil || len(response.Splits) != 0 {
		t.Fatalf("expected empty splits slice, got %#v", response.Splits)
	}
}
