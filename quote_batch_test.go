package twelvedata

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQuoteBatchDecodesQuotesPerItemErrorsAndMissingSymbols(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/quote" {
			t.Fatalf("expected /quote, got %s", r.URL.Path)
		}
		if actual := r.URL.Query().Get("symbol"); actual != "AAPL,TSLA,MISSING" {
			t.Fatalf("unexpected symbols: %q", actual)
		}
		if actual := r.URL.Query().Get("mic_code"); actual != "XNGS" {
			t.Fatalf("unexpected MIC: %q", actual)
		}
		if actual := r.URL.Query().Get("exchange"); actual != "NASDAQ" {
			t.Fatalf("unexpected exchange: %q", actual)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"AAPL": map[string]any{
				"symbol": "AAPL", "mic_code": "XNGS", "close": "213.50",
			},
			"TSLA": map[string]any{
				"status": "error", "code": 404, "message": "symbol unavailable",
			},
		})
	}))
	t.Cleanup(server.Close)

	client := NewClient("demo", WithBaseURL(server.URL))
	request, err := client.QuoteBatch(QuoteBatchParams{
		Symbols:  []string{" AAPL ", "TSLA", "MISSING"},
		MICCode:  "XNGS",
		Exchange: "NASDAQ",
		Interval: "1day",
	})
	if err != nil {
		t.Fatalf("create quote batch: %v", err)
	}
	response, err := request.AsJSON(context.Background())
	if err != nil {
		t.Fatalf("execute quote batch: %v", err)
	}
	if response.Items["AAPL"].Quote == nil ||
		response.Items["AAPL"].Quote.Close != "213.50" {
		t.Fatalf("unexpected AAPL quote: %#v", response.Items["AAPL"])
	}
	if response.Items["TSLA"].Error == nil ||
		response.Items["TSLA"].Error.Code != 404 {
		t.Fatalf("unexpected TSLA error: %#v", response.Items["TSLA"])
	}
	if response.Items["MISSING"].Error == nil ||
		response.Items["MISSING"].Error.Message != "quote missing from batch response" {
		t.Fatalf("unexpected missing-symbol result: %#v", response.Items["MISSING"])
	}
}

func TestQuoteBatchSupportsWrappedDataAndTopLevelErrors(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount++
		if requestCount == 1 {
			_, _ = w.Write([]byte(`{
				"status":"ok",
				"data":{
					"AAPL":{"symbol":"AAPL","close":"213.50"},
					"TSLA":{"symbol":"TSLA","close":"316.10"}
				}
			}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"error","code":429,"message":"quota exceeded"}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient("demo", WithBaseURL(server.URL))
	first, err := client.QuoteBatch(QuoteBatchParams{Symbols: []string{"AAPL", "TSLA"}})
	if err != nil {
		t.Fatalf("create wrapped quote batch: %v", err)
	}
	response, err := first.AsJSON(context.Background())
	if err != nil || response.Items["TSLA"].Quote == nil {
		t.Fatalf("unexpected wrapped response: %#v err=%v", response, err)
	}

	second, err := client.QuoteBatch(QuoteBatchParams{Symbols: []string{"AAPL", "TSLA"}})
	if err != nil {
		t.Fatalf("create error quote batch: %v", err)
	}
	_, err = second.AsJSON(context.Background())
	var apiError *APIError
	if !errors.As(err, &apiError) || apiError.Code != 429 {
		t.Fatalf("expected top-level API error, got %T %v", err, err)
	}
}

func TestQuoteBatchDecodesSingleObjectAndRejectsEmptyOrMalformedItems(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount++
		switch requestCount {
		case 1:
			_, _ = w.Write([]byte(`{"symbol":"AAPL","mic_code":"XNGS","close":"213.50"}`))
		case 2:
			_, _ = w.Write([]byte(`{"AAPL":{},"TSLA":"invalid"}`))
		}
	}))
	t.Cleanup(server.Close)
	client := NewClient("demo", WithBaseURL(server.URL))

	first, err := client.QuoteBatch(QuoteBatchParams{Symbols: []string{"AAPL", "TSLA"}})
	if err != nil {
		t.Fatalf("create single-object batch request: %v", err)
	}
	response, err := first.AsJSON(context.Background())
	if err != nil {
		t.Fatalf("decode single-object response: %v", err)
	}
	if response.Items["AAPL"].Quote == nil ||
		response.Items["TSLA"].Error == nil {
		t.Fatalf("unexpected single-object mapping: %#v", response.Items)
	}

	second, err := client.QuoteBatch(QuoteBatchParams{Symbols: []string{"AAPL", "TSLA"}})
	if err != nil {
		t.Fatalf("create invalid-item batch request: %v", err)
	}
	response, err = second.AsJSON(context.Background())
	if err != nil {
		t.Fatalf("decode invalid items: %v", err)
	}
	if response.Items["AAPL"].Error == nil ||
		response.Items["TSLA"].Error == nil {
		t.Fatalf("expected empty and malformed item failures: %#v", response.Items)
	}
}

func TestQuoteBatchValidatesSymbols(t *testing.T) {
	t.Parallel()

	client := NewClient("demo")
	for name, symbols := range map[string][]string{
		"one symbol":       {"AAPL"},
		"duplicate symbol": {"AAPL", "aapl", "TSLA"},
		"blank symbol":     {"AAPL", " "},
		"embedded comma":   {"AAPL", "TSLA,NVDA"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := client.QuoteBatch(QuoteBatchParams{Symbols: symbols}); err == nil {
				t.Fatal("expected quote batch validation error")
			}
		})
	}
}
