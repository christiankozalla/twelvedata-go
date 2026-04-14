package twelvedata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func boolPtr(v bool) *bool {
	value := v
	return &value
}

func intPtr(v int) *int {
	value := v
	return &value
}

func floatPtr(v float64) *float64 {
	value := v
	return &value
}

func TestListEndpointsRequests(t *testing.T) {
	tests := []struct {
		name         string
		build        func(*Client) *Request
		expectedPath string
		expected     map[string]string
	}{
		{
			name: "stocks list",
			build: func(c *Client) *Request {
				return c.StocksList(StocksListParams{Exchange: "NASDAQ"})
			},
			expectedPath: "/stocks",
			expected:     map[string]string{"exchange": "NASDAQ"},
		},
		{
			name: "stock exchanges list",
			build: func(c *Client) *Request {
				return c.StockExchangesList()
			},
			expectedPath: "/stock_exchanges",
			expected:     map[string]string{},
		},
		{
			name: "forex pairs list",
			build: func(c *Client) *Request {
				return c.ForexPairsList(ForexPairsListParams{CurrencyBase: "EUR", CurrencyQuote: "USD"})
			},
			expectedPath: "/forex_pairs",
			expected: map[string]string{
				"currency_base":  "EUR",
				"currency_quote": "USD",
			},
		},
		{
			name: "cryptocurrencies list",
			build: func(c *Client) *Request {
				return c.CryptocurrenciesList(CryptocurrenciesListParams{Exchange: "BINANCE"})
			},
			expectedPath: "/cryptocurrencies",
			expected:     map[string]string{"exchange": "BINANCE"},
		},
		{
			name: "etf list",
			build: func(c *Client) *Request {
				return c.ETFList(ETFListParams{Country: "US", IncludeDelisted: boolPtr(true)})
			},
			expectedPath: "/etf",
			expected: map[string]string{
				"country":          "US",
				"include_delisted": "true",
			},
		},
		{
			name: "indices list",
			build: func(c *Client) *Request {
				return c.IndicesList(IndicesListParams{MICCode: "XNYS"})
			},
			expectedPath: "/indices",
			expected:     map[string]string{"mic_code": "XNYS"},
		},
		{
			name: "funds list",
			build: func(c *Client) *Request {
				return c.FundsList(FundsListParams{Page: intPtr(2), OutputSize: intPtr(25)})
			},
			expectedPath: "/funds",
			expected: map[string]string{
				"page":       "2",
				"outputsize": "25",
			},
		},
		{
			name: "bonds list",
			build: func(c *Client) *Request {
				return c.BondsList(BondsListParams{Type: "Corporate", ShowPlan: boolPtr(false)})
			},
			expectedPath: "/bonds",
			expected: map[string]string{
				"type":      "Corporate",
				"show_plan": "false",
			},
		},
		{
			name: "exchanges list",
			build: func(c *Client) *Request {
				return c.ExchangesList(ExchangesListParams{Name: "New York", ShowPlan: boolPtr(true)})
			},
			expectedPath: "/exchanges",
			expected: map[string]string{
				"name":      "New York",
				"show_plan": "true",
			},
		},
		{
			name: "technical indicators list",
			build: func(c *Client) *Request {
				return c.TechnicalIndicatorsList()
			},
			expectedPath: "/technical_indicators",
			expected:     map[string]string{},
		},
		{
			name: "symbol search",
			build: func(c *Client) *Request {
				return c.SymbolSearch(SymbolSearchParams{Symbol: "AAPL", OutputSize: intPtr(10), ShowPlan: boolPtr(true)})
			},
			expectedPath: "/symbol_search",
			expected: map[string]string{
				"symbol":     "AAPL",
				"outputsize": "10",
				"show_plan":  "true",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var capturedPath string
			var capturedQuery url.Values

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				capturedQuery = r.URL.Query()
				w.Header().Set("Content-Type", "application/json")
				payload := map[string]any{
					"data": []map[string]string{{"symbol": "AAPL"}},
				}
				_ = json.NewEncoder(w).Encode(payload)
			}))
			defer server.Close()

			client := NewClient("demo", WithBaseURL(server.URL))
			req := tt.build(client)

			builtURL, err := req.AsURL()
			if err != nil {
				t.Fatalf("AsURL: %v", err)
			}

			parsed, err := url.Parse(builtURL)
			if err != nil {
				t.Fatalf("parse url: %v", err)
			}
			if parsed.Path != tt.expectedPath {
				t.Fatalf("expected path %q, got %q", tt.expectedPath, parsed.Path)
			}
			query := parsed.Query()
			if query.Get("apikey") != "demo" {
				t.Fatalf("expected apikey=demo, got %q", query.Get("apikey"))
			}
			if query.Get("format") != "JSON" {
				t.Fatalf("expected format=JSON, got %q", query.Get("format"))
			}
			if query.Get("source") != requestSource {
				t.Fatalf("expected source=%s, got %q", requestSource, query.Get("source"))
			}
			for key, want := range tt.expected {
				if got := query.Get(key); got != want {
					t.Fatalf("expected query %s=%s, got %s", key, want, got)
				}
			}

			payload := map[string]any{}
			if err := req.AsJSON(context.Background(), &payload); err != nil {
				t.Fatalf("AsJSON: %v", err)
			}
			if payload == nil {
				t.Fatalf("expected payload, got nil")
			}
			if capturedPath != tt.expectedPath {
				t.Fatalf("server saw path %q, want %q", capturedPath, tt.expectedPath)
			}
			if capturedQuery == nil {
				t.Fatalf("server query nil")
			}
			if capturedQuery.Get("apikey") != "demo" {
				t.Fatalf("server query apikey=demo expected, got %q", capturedQuery.Get("apikey"))
			}
			if capturedQuery.Get("format") != "JSON" {
				t.Fatalf("server query format=JSON expected, got %q", capturedQuery.Get("format"))
			}
			if capturedQuery.Get("source") != requestSource {
				t.Fatalf("server query source=%s expected, got %q", requestSource, capturedQuery.Get("source"))
			}
			for key, want := range tt.expected {
				if got := capturedQuery.Get(key); got != want {
					t.Fatalf("server query %s=%s expected, got %s", key, want, got)
				}
			}
		})
	}
}

func TestRequestAsCSV(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("format"); got != "CSV" {
			t.Fatalf("expected format=CSV, got %s", got)
		}
		if got := r.URL.Query().Get("apikey"); got != "demo" {
			t.Fatalf("expected apikey=demo, got %s", got)
		}
		if got := r.URL.Query().Get("source"); got != requestSource {
			t.Fatalf("expected source=%s, got %s", requestSource, got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("symbol\nAAPL\n"))
	}))
	defer server.Close()

	client := NewClient("demo", WithBaseURL(server.URL))
	csv, err := client.StocksList(StocksListParams{}).AsCSV(context.Background())
	if err != nil {
		t.Fatalf("AsCSV: %v", err)
	}
	want := "symbol\nAAPL\n"
	if csv != want {
		t.Fatalf("expected csv %q, got %q", want, csv)
	}
}

func TestDataEndpointsNormalization(t *testing.T) {
	tests := []struct {
		name         string
		build        func(*Client) *Request
		expectedPath string
		expected     map[string]string
		response     map[string]any
		assert       func(*testing.T, interface{})
	}{
		{
			name: "exchange rate",
			build: func(c *Client) *Request {
				return c.ExchangeRate(ExchangeRateParams{Symbol: "EUR/USD", DP: intPtr(4)})
			},
			expectedPath: "/exchange_rate",
			expected: map[string]string{
				"symbol": "EUR/USD",
				"dp":     "4",
			},
			response: map[string]any{
				"status": "ok",
				"data": map[string]any{
					"symbol": "EUR/USD",
					"rate":   "1.2345",
				},
			},
			assert: func(t *testing.T, data interface{}) {
				m, ok := data.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map, got %T", data)
				}
				if m["symbol"] != "EUR/USD" {
					t.Fatalf("expected symbol EUR/USD, got %v", m["symbol"])
				}
				if m["rate"] != "1.2345" {
					t.Fatalf("expected rate 1.2345, got %v", m["rate"])
				}
			},
		},
		{
			name: "currency conversion",
			build: func(c *Client) *Request {
				return c.CurrencyConversion(CurrencyConversionParams{Symbol: "EUR/USD", Amount: floatPtr(100.5)})
			},
			expectedPath: "/currency_conversion",
			expected: map[string]string{
				"symbol": "EUR/USD",
				"amount": "100.5",
			},
			response: map[string]any{
				"status": "ok",
				"data": map[string]any{
					"symbol":    "EUR/USD",
					"amount":    "100.5",
					"converted": "105.2",
				},
			},
			assert: func(t *testing.T, data interface{}) {
				m, ok := data.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map, got %T", data)
				}
				if m["converted"] != "105.2" {
					t.Fatalf("expected converted 105.2, got %v", m["converted"])
				}
			},
		},
		{
			name: "quote",
			build: func(c *Client) *Request {
				return c.Quote(QuoteParams{
					Symbol:           "AAPL",
					Interval:         "1day",
					VolumeTimePeriod: "10day",
					Type:             "Common",
					DP:               intPtr(3),
					Timezone:         "America/New_York",
					Prepost:          boolPtr(true),
					MICCode:          "XNAS",
					EOD:              "true",
					RollingPeriod:    "30day",
				})
			},
			expectedPath: "/quote",
			expected: map[string]string{
				"symbol":             "AAPL",
				"interval":           "1day",
				"volume_time_period": "10day",
				"type":               "Common",
				"dp":                 "3",
				"timezone":           "America/New_York",
				"prepost":            "true",
				"mic_code":           "XNAS",
				"eod":                "true",
				"rolling_period":     "30day",
			},
			response: map[string]any{
				"status": "ok",
				"data": map[string]any{
					"symbol": "AAPL",
					"open":   "150.00",
				},
			},
			assert: func(t *testing.T, data interface{}) {
				m, ok := data.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map, got %T", data)
				}
				if m["open"] != "150.00" {
					t.Fatalf("expected open 150.00, got %v", m["open"])
				}
			},
		},
		{
			name: "indicator rsi",
			build: func(c *Client) *Request {
				return c.Indicator("rsi", IndicatorParams{
					Symbol:     "AAPL",
					Interval:   "1day",
					SeriesType: "close",
					TimePeriod: intPtr(14),
				})
			},
			expectedPath: "/rsi",
			expected: map[string]string{
				"symbol":      "AAPL",
				"interval":    "1day",
				"series_type": "close",
				"time_period": "14",
			},
			response: map[string]any{
				"values": []map[string]any{
					{
						"datetime": "2024-01-01",
						"rsi":      "45.10",
					},
				},
			},
			assert: func(t *testing.T, data interface{}) {
				slice, ok := data.([]interface{})
				if !ok {
					t.Fatalf("expected slice, got %T", data)
				}
				if len(slice) != 1 {
					t.Fatalf("expected 1 value, got %d", len(slice))
				}
				entry, ok := slice[0].(map[string]interface{})
				if !ok {
					t.Fatalf("expected map entry, got %T", slice[0])
				}
				if entry["rsi"] != "45.10" {
					t.Fatalf("expected RSI 45.10, got %v", entry["rsi"])
				}
			},
		},
		{
			name: "willr",
			build: func(c *Client) *Request {
				return c.WILLR(WILLRParams{
					Symbol:        "AAPL",
					Interval:      "1day",
					TimePeriod:    intPtr(14),
					OutputSize:    intPtr(5),
					Timezone:      "Exchange",
					Order:         "desc",
					IncludeOHLC:   boolPtr(true),
					PreviousClose: boolPtr(true),
				})
			},
			expectedPath: "/willr",
			expected: map[string]string{
				"symbol":         "AAPL",
				"interval":       "1day",
				"time_period":    "14",
				"outputsize":     "5",
				"timezone":       "Exchange",
				"order":          "desc",
				"include_ohlc":   "true",
				"previous_close": "true",
			},
			response: map[string]any{
				"meta": map[string]any{
					"symbol":   "AAPL",
					"interval": "1day",
					"indicator": map[string]any{
						"name":        "WILLR - Williams %R",
						"time_period": 14,
					},
				},
				"values": []map[string]any{
					{
						"datetime": "2024-01-01",
						"willr":    "-84.8916",
					},
				},
				"status": "ok",
			},
			assert: func(t *testing.T, data interface{}) {
				slice, ok := data.([]interface{})
				if !ok {
					t.Fatalf("expected slice, got %T", data)
				}
				if len(slice) != 1 {
					t.Fatalf("expected 1 value, got %d", len(slice))
				}
				entry, ok := slice[0].(map[string]interface{})
				if !ok {
					t.Fatalf("expected map entry, got %T", slice[0])
				}
				if entry["willr"] != "-84.8916" {
					t.Fatalf("expected WILLR -84.8916, got %v", entry["willr"])
				}
			},
		},
		{
			name: "adx",
			build: func(c *Client) *Request {
				return c.ADX(ADXParams{
					Symbol:        "AAPL",
					Interval:      "1day",
					TimePeriod:    intPtr(14),
					OutputSize:    intPtr(5),
					Timezone:      "Exchange",
					Order:         "desc",
					IncludeOHLC:   boolPtr(true),
					PreviousClose: boolPtr(true),
				})
			},
			expectedPath: "/adx",
			expected: map[string]string{
				"symbol":         "AAPL",
				"interval":       "1day",
				"time_period":    "14",
				"outputsize":     "5",
				"timezone":       "Exchange",
				"order":          "desc",
				"include_ohlc":   "true",
				"previous_close": "true",
			},
			response: map[string]any{
				"meta": map[string]any{
					"symbol":   "AAPL",
					"interval": "1day",
					"indicator": map[string]any{
						"name":        "ADX - Average Directional Index",
						"time_period": 14,
					},
				},
				"values": []map[string]any{
					{
						"datetime": "2024-01-01",
						"adx":      "49.22897",
					},
				},
				"status": "ok",
			},
			assert: func(t *testing.T, data interface{}) {
				slice, ok := data.([]interface{})
				if !ok {
					t.Fatalf("expected slice, got %T", data)
				}
				if len(slice) != 1 {
					t.Fatalf("expected 1 value, got %d", len(slice))
				}
				entry, ok := slice[0].(map[string]interface{})
				if !ok {
					t.Fatalf("expected map entry, got %T", slice[0])
				}
				if entry["adx"] != "49.22897" {
					t.Fatalf("expected ADX 49.22897, got %v", entry["adx"])
				}
			},
		},
		{
			name: "plus_di",
			build: func(c *Client) *Request {
				return c.PlusDI(PlusDIParams{
					Symbol:        "AAPL",
					Interval:      "1day",
					TimePeriod:    intPtr(9),
					OutputSize:    intPtr(5),
					Timezone:      "Exchange",
					Order:         "desc",
					IncludeOHLC:   boolPtr(true),
					PreviousClose: boolPtr(true),
				})
			},
			expectedPath: "/plus_di",
			expected: map[string]string{
				"symbol":         "AAPL",
				"interval":       "1day",
				"time_period":    "9",
				"outputsize":     "5",
				"timezone":       "Exchange",
				"order":          "desc",
				"include_ohlc":   "true",
				"previous_close": "true",
			},
			response: map[string]any{
				"meta": map[string]any{
					"symbol":   "AAPL",
					"interval": "1day",
					"indicator": map[string]any{
						"name":        "PLUS_DI - Plus Directional Indicator",
						"time_period": 9,
					},
				},
				"values": []map[string]any{
					{
						"datetime": "2024-01-01",
						"plus_di":  "7.69578",
					},
				},
				"status": "ok",
			},
			assert: func(t *testing.T, data interface{}) {
				slice, ok := data.([]interface{})
				if !ok {
					t.Fatalf("expected slice, got %T", data)
				}
				if len(slice) != 1 {
					t.Fatalf("expected 1 value, got %d", len(slice))
				}
				entry, ok := slice[0].(map[string]interface{})
				if !ok {
					t.Fatalf("expected map entry, got %T", slice[0])
				}
				if entry["plus_di"] != "7.69578" {
					t.Fatalf("expected PLUS_DI 7.69578, got %v", entry["plus_di"])
				}
			},
		},
		{
			name: "minus_di",
			build: func(c *Client) *Request {
				return c.MinusDI(MinusDIParams{
					Symbol:        "AAPL",
					Interval:      "1day",
					TimePeriod:    intPtr(9),
					OutputSize:    intPtr(5),
					Timezone:      "Exchange",
					Order:         "desc",
					IncludeOHLC:   boolPtr(true),
					PreviousClose: boolPtr(true),
				})
			},
			expectedPath: "/minus_di",
			expected: map[string]string{
				"symbol":         "AAPL",
				"interval":       "1day",
				"time_period":    "9",
				"outputsize":     "5",
				"timezone":       "Exchange",
				"order":          "desc",
				"include_ohlc":   "true",
				"previous_close": "true",
			},
			response: map[string]any{
				"meta": map[string]any{
					"symbol":   "AAPL",
					"interval": "1day",
					"indicator": map[string]any{
						"name":        "MINUS_DI - Minus Directional Indicator",
						"time_period": 9,
					},
				},
				"values": []map[string]any{
					{
						"datetime": "2024-01-01",
						"minus_di": "46.60579",
					},
				},
				"status": "ok",
			},
			assert: func(t *testing.T, data interface{}) {
				slice, ok := data.([]interface{})
				if !ok {
					t.Fatalf("expected slice, got %T", data)
				}
				if len(slice) != 1 {
					t.Fatalf("expected 1 value, got %d", len(slice))
				}
				entry, ok := slice[0].(map[string]interface{})
				if !ok {
					t.Fatalf("expected map entry, got %T", slice[0])
				}
				if entry["minus_di"] != "46.60579" {
					t.Fatalf("expected MINUS_DI 46.60579, got %v", entry["minus_di"])
				}
			},
		},
		{
			name: "price",
			build: func(c *Client) *Request {
				return c.Price(PriceParams{Symbol: "AAPL", Prepost: boolPtr(false)})
			},
			expectedPath: "/price",
			expected: map[string]string{
				"symbol":  "AAPL",
				"prepost": "false",
			},
			response: map[string]any{
				"status": "ok",
				"data": map[string]any{
					"price": "152.10",
				},
			},
			assert: func(t *testing.T, data interface{}) {
				m, ok := data.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map, got %T", data)
				}
				if m["price"] != "152.10" {
					t.Fatalf("expected price 152.10, got %v", m["price"])
				}
			},
		},
		{
			name: "eod",
			build: func(c *Client) *Request {
				return c.EOD(EODParams{Symbol: "AAPL", Date: "2024-01-01"})
			},
			expectedPath: "/eod",
			expected: map[string]string{
				"symbol": "AAPL",
				"date":   "2024-01-01",
			},
			response: map[string]any{
				"status": "ok",
				"data": map[string]any{
					"close": "148.90",
				},
			},
			assert: func(t *testing.T, data interface{}) {
				m, ok := data.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map, got %T", data)
				}
				if m["close"] != "148.90" {
					t.Fatalf("expected close 148.90, got %v", m["close"])
				}
			},
		},
		{
			name: "options expiration",
			build: func(c *Client) *Request {
				return c.OptionsExpiration(OptionsExpirationParams{Symbol: "AAPL", MICCode: "XNAS"})
			},
			expectedPath: "/options/expiration",
			expected: map[string]string{
				"symbol":   "AAPL",
				"mic_code": "XNAS",
			},
			response: map[string]any{
				"status": "ok",
				"data": map[string]any{
					"dates": []any{"2024-01-19"},
				},
			},
			assert: func(t *testing.T, data interface{}) {
				m, ok := data.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map, got %T", data)
				}
				dates, ok := m["dates"].([]interface{})
				if !ok || len(dates) == 0 {
					t.Fatalf("expected non-empty dates, got %v", m["dates"])
				}
				if dates[0] != "2024-01-19" {
					t.Fatalf("expected first date 2024-01-19, got %v", dates[0])
				}
			},
		},
		{
			name: "options chain",
			build: func(c *Client) *Request {
				return c.OptionsChain(OptionsChainParams{Symbol: "AAPL", ExpirationDate: "2024-01-19", Side: "call"})
			},
			expectedPath: "/options/chain",
			expected: map[string]string{
				"symbol":          "AAPL",
				"expiration_date": "2024-01-19",
				"side":            "call",
			},
			response: map[string]any{
				"status": "ok",
				"data": map[string]any{
					"calls": []any{map[string]any{"strike": "150"}},
				},
			},
			assert: func(t *testing.T, data interface{}) {
				m, ok := data.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map, got %T", data)
				}
				calls, ok := m["calls"].([]interface{})
				if !ok || len(calls) == 0 {
					t.Fatalf("expected calls array, got %v", m["calls"])
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var capturedPath string
			var capturedQuery url.Values

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				capturedQuery = r.URL.Query()
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := NewClient("demo", WithBaseURL(server.URL))
			req := tt.build(client)

			builtURL, err := req.AsURL()
			if err != nil {
				t.Fatalf("AsURL: %v", err)
			}

			parsed, err := url.Parse(builtURL)
			if err != nil {
				t.Fatalf("parse url: %v", err)
			}
			if parsed.Path != tt.expectedPath {
				t.Fatalf("expected path %q, got %q", tt.expectedPath, parsed.Path)
			}

			normalized, err := req.AsNormalizedJSON(context.Background())
			if err != nil {
				t.Fatalf("AsNormalizedJSON: %v", err)
			}
			tt.assert(t, normalized)

			if capturedPath != tt.expectedPath {
				t.Fatalf("server path %q, want %q", capturedPath, tt.expectedPath)
			}
			if capturedQuery == nil {
				t.Fatalf("server query nil")
			}
			if capturedQuery.Get("apikey") != "demo" {
				t.Fatalf("server query apikey=demo expected, got %q", capturedQuery.Get("apikey"))
			}
			if capturedQuery.Get("format") != "JSON" {
				t.Fatalf("server query format=JSON expected, got %q", capturedQuery.Get("format"))
			}
			if capturedQuery.Get("source") != requestSource {
				t.Fatalf("server query source=%s expected, got %q", requestSource, capturedQuery.Get("source"))
			}
			for key, want := range tt.expected {
				if got := capturedQuery.Get(key); got != want {
					t.Fatalf("server query %s=%s expected, got %s", key, want, got)
				}
			}
		})
	}
}

func TestTimeSeriesRequests(t *testing.T) {
	var queries []url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		if r.URL.Query().Get("format") == "CSV" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("datetime,open\n2024-01-01 09:30:00,150.00\n"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		payload := map[string]any{
			"status": "ok",
			"values": []map[string]any{
				{
					"datetime": "2024-01-01 09:30:00",
					"open":     "150.00",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer server.Close()

	client := NewClient("demo", WithBaseURL(server.URL))
	ts := client.TimeSeries(TimeSeriesParams{Symbol: "AAPL", Interval: "1min", OutputSize: intPtr(1)})

	series, err := ts.AsJSON(context.Background())
	if err != nil {
		t.Fatalf("AsJSON: %v", err)
	}
	if len(series) != 1 {
		t.Fatalf("expected 1 datapoint, got %d", len(series))
	}
	if series[0]["open"] != "150.00" {
		t.Fatalf("expected open 150.00, got %v", series[0]["open"])
	}

	urls, err := ts.AsURL()
	if err != nil {
		t.Fatalf("AsURL: %v", err)
	}
	if len(urls) != 1 {
		t.Fatalf("expected 1 url, got %d", len(urls))
	}
	parsedURL, err := url.Parse(urls[0])
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	if parsedURL.Path != "/time_series" {
		t.Fatalf("expected /time_series path, got %s", parsedURL.Path)
	}

	records, err := ts.AsCSV(context.Background())
	if err != nil {
		t.Fatalf("AsCSV: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 csv rows, got %d", len(records))
	}
	if records[1][0] != "2024-01-01 09:30:00" {
		t.Fatalf("expected csv datetime 2024-01-01 09:30:00, got %s", records[1][0])
	}

	if len(queries) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(queries))
	}
	first := queries[0]
	if first.Get("symbol") != "AAPL" {
		t.Fatalf("expected first request symbol AAPL, got %s", first.Get("symbol"))
	}
	if first.Get("format") != "JSON" {
		t.Fatalf("expected first request format JSON, got %s", first.Get("format"))
	}
	if first.Get("source") != requestSource {
		t.Fatalf("expected first request source %s, got %s", requestSource, first.Get("source"))
	}
	second := queries[1]
	if second.Get("format") != "CSV" {
		t.Fatalf("expected second request format CSV, got %s", second.Get("format"))
	}
	if second.Get("source") != requestSource {
		t.Fatalf("expected second request source %s, got %s", requestSource, second.Get("source"))
	}
}

func TestTimeSeriesTypedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"meta": map[string]any{
				"symbol":            "AAPL",
				"interval":          "1day",
				"currency":          "USD",
				"exchange_timezone": "America/New_York",
				"exchange":          "NASDAQ",
				"mic_code":          "XNAS",
				"type":              "Common Stock",
			},
			"values": []map[string]any{
				{
					"datetime": "2024-01-01",
					"open":     "150.00",
					"high":     "155.00",
					"low":      "149.50",
					"close":    "152.10",
					"volume":   "123456",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient("demo", WithBaseURL(server.URL))
	response, err := client.TimeSeries(TimeSeriesParams{Symbol: "AAPL", Interval: "1day"}).AsResponse(context.Background())
	if err != nil {
		t.Fatalf("AsResponse: %v", err)
	}
	if response.Meta.Symbol != "AAPL" {
		t.Fatalf("expected symbol AAPL, got %q", response.Meta.Symbol)
	}
	if response.Meta.ExchangeTimezone != "America/New_York" {
		t.Fatalf("expected exchange timezone America/New_York, got %q", response.Meta.ExchangeTimezone)
	}
	if len(response.Values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(response.Values))
	}
	if response.Values[0].Close != "152.10" {
		t.Fatalf("expected close 152.10, got %q", response.Values[0].Close)
	}
}

func TestRequestAsRawJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","data":{"symbol":"AAPL"}}`))
	}))
	defer server.Close()

	client := NewClient("demo", WithBaseURL(server.URL))
	payload, err := client.Price(PriceParams{Symbol: "AAPL"}).AsRawJSON(context.Background())
	if err != nil {
		t.Fatalf("AsRawJSON: %v", err)
	}
	if string(payload) != `{"status":"ok","data":{"symbol":"AAPL"}}` {
		t.Fatalf("unexpected raw payload %s", string(payload))
	}
}

func TestRequestAsNormalizedIntoTypedPriceResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"data": map[string]any{
				"symbol": "AAPL",
				"price":  "152.10",
			},
		})
	}))
	defer server.Close()

	client := NewClient("demo", WithBaseURL(server.URL))
	var response PriceResponse
	err := client.Price(PriceParams{Symbol: "AAPL"}).AsNormalized(context.Background(), &response)
	if err != nil {
		t.Fatalf("AsNormalized: %v", err)
	}
	if response.Symbol != "AAPL" {
		t.Fatalf("expected symbol AAPL, got %q", response.Symbol)
	}
	if response.Price != "152.10" {
		t.Fatalf("expected price 152.10, got %q", response.Price)
	}
}

func TestRequestAsNormalizedIntoTypedQuoteResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"data": map[string]any{
				"symbol":         "AAPL",
				"exchange":       "NASDAQ",
				"mic_code":       "XNAS",
				"open":           "150.00",
				"close":          "152.10",
				"is_market_open": true,
				"fifty_two_week": map[string]any{
					"low":  "120.00",
					"high": "199.00",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient("demo", WithBaseURL(server.URL))
	var response QuoteResponse
	err := client.Quote(QuoteParams{Symbol: "AAPL"}).AsNormalized(context.Background(), &response)
	if err != nil {
		t.Fatalf("AsNormalized: %v", err)
	}
	if response.Symbol != "AAPL" {
		t.Fatalf("expected symbol AAPL, got %q", response.Symbol)
	}
	if response.Exchange != "NASDAQ" {
		t.Fatalf("expected exchange NASDAQ, got %q", response.Exchange)
	}
	if !response.IsMarketOpen {
		t.Fatal("expected IsMarketOpen to be true")
	}
	if response.FiftyTwoWeek.High != "199.00" {
		t.Fatalf("expected 52-week high 199.00, got %q", response.FiftyTwoWeek.High)
	}
}

func TestWILLRTypedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"meta": map[string]any{
				"symbol":            "AAPL",
				"interval":          "1day",
				"currency":          "USD",
				"exchange_timezone": "America/New_York",
				"exchange":          "NASDAQ",
				"mic_code":          "XNAS",
				"type":              "Common Stock",
				"indicator": map[string]any{
					"name":        "WILLR - Williams %R",
					"time_period": 14,
				},
			},
			"values": []map[string]any{
				{
					"datetime": "2024-01-01",
					"willr":    "-84.8916",
					"open":     "150.00",
					"high":     "155.00",
					"low":      "149.50",
					"close":    "152.10",
				},
			},
			"status": "ok",
		})
	}))
	defer server.Close()

	client := NewClient("demo", WithBaseURL(server.URL))
	var response WILLRResponse
	err := client.WILLR(WILLRParams{Symbol: "AAPL", Interval: "1day", TimePeriod: intPtr(14)}).AsJSON(context.Background(), &response)
	if err != nil {
		t.Fatalf("AsJSON: %v", err)
	}
	if response.Meta.Symbol != "AAPL" {
		t.Fatalf("expected symbol AAPL, got %q", response.Meta.Symbol)
	}
	if response.Meta.Indicator.TimePeriod != 14 {
		t.Fatalf("expected time period 14, got %d", response.Meta.Indicator.TimePeriod)
	}
	if len(response.Values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(response.Values))
	}
	if response.Values[0].WILLR != "-84.8916" {
		t.Fatalf("expected WILLR -84.8916, got %q", response.Values[0].WILLR)
	}
}

func TestADXTypedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"meta": map[string]any{
				"symbol":            "AAPL",
				"interval":          "1day",
				"currency":          "USD",
				"exchange_timezone": "America/New_York",
				"exchange":          "NASDAQ",
				"mic_code":          "XNAS",
				"type":              "Common Stock",
				"indicator": map[string]any{
					"name":        "ADX - Average Directional Index",
					"time_period": 14,
				},
			},
			"values": []map[string]any{
				{
					"datetime": "2024-01-01",
					"adx":      "49.22897",
					"open":     "150.00",
					"high":     "155.00",
					"low":      "149.50",
					"close":    "152.10",
				},
			},
			"status": "ok",
		})
	}))
	defer server.Close()

	client := NewClient("demo", WithBaseURL(server.URL))
	var response ADXResponse
	err := client.ADX(ADXParams{Symbol: "AAPL", Interval: "1day", TimePeriod: intPtr(14)}).AsJSON(context.Background(), &response)
	if err != nil {
		t.Fatalf("AsJSON: %v", err)
	}
	if response.Meta.Symbol != "AAPL" {
		t.Fatalf("expected symbol AAPL, got %q", response.Meta.Symbol)
	}
	if response.Meta.Indicator.TimePeriod != 14 {
		t.Fatalf("expected time period 14, got %d", response.Meta.Indicator.TimePeriod)
	}
	if len(response.Values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(response.Values))
	}
	if response.Values[0].ADX != "49.22897" {
		t.Fatalf("expected ADX 49.22897, got %q", response.Values[0].ADX)
	}
}

func TestPlusDITypedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"meta": map[string]any{
				"symbol":            "AAPL",
				"interval":          "1day",
				"currency":          "USD",
				"exchange_timezone": "America/New_York",
				"exchange":          "NASDAQ",
				"mic_code":          "XNAS",
				"type":              "Common Stock",
				"indicator": map[string]any{
					"name":        "PLUS_DI - Plus Directional Indicator",
					"time_period": 9,
				},
			},
			"values": []map[string]any{
				{
					"datetime": "2024-01-01",
					"plus_di":  "7.69578",
				},
			},
			"status": "ok",
		})
	}))
	defer server.Close()

	client := NewClient("demo", WithBaseURL(server.URL))
	var response PlusDIResponse
	err := client.PlusDI(PlusDIParams{Symbol: "AAPL", Interval: "1day", TimePeriod: intPtr(9)}).AsJSON(context.Background(), &response)
	if err != nil {
		t.Fatalf("AsJSON: %v", err)
	}
	if response.Meta.Symbol != "AAPL" {
		t.Fatalf("expected symbol AAPL, got %q", response.Meta.Symbol)
	}
	if response.Meta.Indicator.TimePeriod != 9 {
		t.Fatalf("expected time period 9, got %d", response.Meta.Indicator.TimePeriod)
	}
	if len(response.Values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(response.Values))
	}
	if response.Values[0].PlusDI != "7.69578" {
		t.Fatalf("expected PLUS_DI 7.69578, got %q", response.Values[0].PlusDI)
	}
}

func TestMinusDITypedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"meta": map[string]any{
				"symbol":            "AAPL",
				"interval":          "1day",
				"currency":          "USD",
				"exchange_timezone": "America/New_York",
				"exchange":          "NASDAQ",
				"mic_code":          "XNAS",
				"type":              "Common Stock",
				"indicator": map[string]any{
					"name":        "MINUS_DI - Minus Directional Indicator",
					"time_period": 9,
				},
			},
			"values": []map[string]any{
				{
					"datetime": "2024-01-01",
					"minus_di": "46.60579",
				},
			},
			"status": "ok",
		})
	}))
	defer server.Close()

	client := NewClient("demo", WithBaseURL(server.URL))
	var response MinusDIResponse
	err := client.MinusDI(MinusDIParams{Symbol: "AAPL", Interval: "1day", TimePeriod: intPtr(9)}).AsJSON(context.Background(), &response)
	if err != nil {
		t.Fatalf("AsJSON: %v", err)
	}
	if response.Meta.Symbol != "AAPL" {
		t.Fatalf("expected symbol AAPL, got %q", response.Meta.Symbol)
	}
	if response.Meta.Indicator.TimePeriod != 9 {
		t.Fatalf("expected time period 9, got %d", response.Meta.Indicator.TimePeriod)
	}
	if len(response.Values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(response.Values))
	}
	if response.Values[0].MinusDI != "46.60579" {
		t.Fatalf("expected MINUS_DI 46.60579, got %q", response.Values[0].MinusDI)
	}
}

func TestRequestReturnsAPIErrorFromJSONBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"error","code":401,"message":"Invalid API key"}`))
	}))
	defer server.Close()

	client := NewClient("bad-key", WithBaseURL(server.URL))
	err := client.Price(PriceParams{Symbol: "AAPL"}).AsJSON(context.Background(), &map[string]any{})
	if err == nil {
		t.Fatal("expected APIError, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != 401 {
		t.Fatalf("expected code 401, got %d", apiErr.Code)
	}
	if apiErr.Message != "Invalid API key" {
		t.Fatalf("expected message Invalid API key, got %q", apiErr.Message)
	}
}

func TestRequestReturnsAPIErrorFromHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"Bad symbol"}`))
	}))
	defer server.Close()

	client := NewClient("demo", WithBaseURL(server.URL))
	_, err := client.Price(PriceParams{Symbol: "BAD"}).AsNormalizedJSON(context.Background())
	if err == nil {
		t.Fatal("expected APIError, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != http.StatusBadRequest {
		t.Fatalf("expected code %d, got %d", http.StatusBadRequest, apiErr.Code)
	}
	if apiErr.Message != "Bad symbol" {
		t.Fatalf("expected message Bad symbol, got %q", apiErr.Message)
	}
}
