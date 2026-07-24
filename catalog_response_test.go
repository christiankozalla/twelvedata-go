package twelvedata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStocksListTypedResponse(t *testing.T) {
	t.Parallel()

	server := newCatalogResponseTestServer(t, `{
		"status": "ok",
		"count": 1,
		"data": [{
			"symbol": "AAPL",
			"name": "Apple Inc.",
			"currency": "USD",
			"exchange": "NASDAQ",
			"mic_code": "XNAS",
			"country": "United States",
			"type": "Common Stock",
			"figi_code": "BBG000B9XRY4",
			"cfi_code": "ESVUFR"
		}]
	}`)

	client := NewClient("demo", WithBaseURL(server.URL))
	var response StocksListResponse
	if err := client.StocksList(StocksListParams{}).AsJSON(context.Background(), &response); err != nil {
		t.Fatalf("decode stocks response: %v", err)
	}

	if response.Status != "ok" || response.Count != 1 || len(response.Data) != 1 {
		t.Fatalf("unexpected stocks response: %#v", response)
	}
	item := response.Data[0]
	if item.MICCode != "XNAS" || item.FIGICode != "BBG000B9XRY4" || item.CFICode != "ESVUFR" {
		t.Fatalf("unexpected stock identifiers: %#v", item)
	}
}

func TestETFListTypedResponse(t *testing.T) {
	t.Parallel()

	server := newCatalogResponseTestServer(t, `{
		"status": "ok",
		"count": 1,
		"data": [{
			"symbol": "SXR8",
			"name": "iShares Core S&P 500 UCITS ETF USD (Acc)",
			"currency": "EUR",
			"exchange": "XETRA",
			"mic_code": "XETR",
			"country": "Germany",
			"type": "ETF",
			"figi_code": "BBG000VRTH18",
			"cfi_code": "CEOGEU"
		}]
	}`)

	client := NewClient("demo", WithBaseURL(server.URL))
	var response ETFListResponse
	if err := client.ETFList(ETFListParams{}).AsJSON(context.Background(), &response); err != nil {
		t.Fatalf("decode ETF response: %v", err)
	}

	if response.Status != "ok" || response.Count != 1 || len(response.Data) != 1 {
		t.Fatalf("unexpected ETF response: %#v", response)
	}
	item := response.Data[0]
	if item.Symbol != "SXR8" || item.MICCode != "XETR" || item.Type != "ETF" {
		t.Fatalf("unexpected ETF item: %#v", item)
	}
}

func TestSymbolSearchTypedResponse(t *testing.T) {
	t.Parallel()

	server := newCatalogResponseTestServer(t, `{
		"status": "ok",
		"data": [{
			"symbol": "AAPL",
			"instrument_name": "Apple Inc",
			"exchange": "NASDAQ",
			"mic_code": "XNAS",
			"instrument_type": "Common Stock",
			"country": "United States",
			"currency": "USD"
		}]
	}`)

	client := NewClient("demo", WithBaseURL(server.URL))
	var response SymbolSearchResponse
	if err := client.SymbolSearch(SymbolSearchParams{Symbol: "AAPL"}).AsJSON(context.Background(), &response); err != nil {
		t.Fatalf("decode symbol-search response: %v", err)
	}

	if response.Status != "ok" || len(response.Data) != 1 {
		t.Fatalf("unexpected symbol-search response: %#v", response)
	}
	item := response.Data[0]
	if item.Symbol != "AAPL" || item.InstrumentName != "Apple Inc" || item.MICCode != "XNAS" {
		t.Fatalf("unexpected symbol-search item: %#v", item)
	}
}

func TestQuoteResponseIncludesLastQuoteAt(t *testing.T) {
	t.Parallel()

	server := newCatalogResponseTestServer(t, `{
		"status": "ok",
		"data": {
			"symbol": "AAPL",
			"timestamp": 1783431000,
			"last_quote_at": 1783454280,
			"close": "312.68"
		}
	}`)

	client := NewClient("demo", WithBaseURL(server.URL))
	var response QuoteResponse
	if err := client.Quote(QuoteParams{Symbol: "AAPL"}).AsNormalized(context.Background(), &response); err != nil {
		t.Fatalf("decode quote response: %v", err)
	}

	if response.Timestamp != 1783431000 || response.LastQuoteAt != 1783454280 {
		t.Fatalf("unexpected quote timestamps: %#v", response)
	}
}

func newCatalogResponseTestServer(t *testing.T, response string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	return server
}
