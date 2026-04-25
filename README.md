# twelvedata-go

`twelvedata-go` is a read-only Go client for the Twelve Data REST API.

To consume Twelve Data API from Go with straightforward request builders and typed response structs for common endpoints.

## Install

```bash
go get github.com/christiankozalla/twelvedata-go
```

## Quick start

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	td "github.com/christiankozalla/twelvedata-go"
)

func main() {
	client := td.NewClient(
		os.Getenv("TWELVEDATA_API_KEY"),
		td.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var price td.PriceResponse
	err := client.Price(td.PriceParams{Symbol: "AAPL"}).AsNormalized(ctx, &price)
	if err != nil {
		var apiErr *td.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("twelvedata api error: %s", apiErr.Error())
		}
		log.Fatal(err)
	}

	fmt.Printf("%s: %s\n", price.Symbol, price.Price)
}
```

## Supported endpoints

- Instrument catalogs: `StocksList`, `StockExchangesList`, `ForexPairsList`, `CryptocurrenciesList`, `ETFList`, `IndicesList`, `FundsList`, `BondsList`, `ExchangesList`, `TechnicalIndicatorsList`
- Discovery: `SymbolSearch`
- Market data: `ExchangeRate`, `CurrencyConversion`, `Quote`, `Price`, `EOD`, `Logo`, `Profile`, `MarketCap`, `Statistics`, `IncomeStatement`, `LastChanges`
- Options: `OptionsExpiration`, `OptionsChain`
- Momentum indicators: `WILLR`, `ADX`, `PlusDI`, `MinusDI`
- Time series builder: `TimeSeries`

## Typed responses

Typed structs are available for the most commonly used endpoints:
- `PriceResponse`
- `QuoteResponse`
- `LogoResponse`
- `ProfileResponse`
- `MarketCapResponse`
- `StatisticsResponse`
- `IncomeStatementResponse`
- `LastChangesResponse`
- `TimeSeriesResponse`
- `WILLRResponse`
- `ADXResponse`
- `PlusDIResponse`
- `MinusDIResponse`

For endpoints without typed structs yet, you can still decode JSON into your own application structs with `AsJSON(...)`.

## Example: Quote

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	td "github.com/christiankozalla/twelvedata-go"
)

func main() {
	client := td.NewClient(os.Getenv("TWELVEDATA_API_KEY"))

	var quote td.QuoteResponse
	err := client.Quote(td.QuoteParams{Symbol: "AAPL"}).AsNormalized(context.Background(), &quote)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s open=%s close=%s\n", quote.Symbol, quote.Open, quote.Close)
}
```

## Example: Time series

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	td "github.com/christiankozalla/twelvedata-go"
)

func main() {
	client := td.NewClient(os.Getenv("TWELVEDATA_API_KEY"))

	series, err := client.TimeSeries(td.TimeSeriesParams{
		Symbol:   "AAPL",
		Interval: "1day",
	}).AsResponse(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	if len(series.Values) == 0 {
		return
	}

	fmt.Println(series.Meta.Symbol, series.Values[0].Datetime, series.Values[0].Close)
}
```

## Example: ADX indicator

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	td "github.com/christiankozalla/twelvedata-go"
)

func main() {
	client := td.NewClient(os.Getenv("TWELVEDATA_API_KEY"))
	timePeriod := 14

	var response td.ADXResponse
	err := client.ADX(td.ADXParams{
		Symbol:     "AAPL",
		Interval:   "1day",
		TimePeriod: &timePeriod,
	}).AsJSON(context.Background(), &response)
	if err != nil {
		log.Fatal(err)
	}

	if len(response.Values) == 0 {
		return
	}

	fmt.Printf("%s %s\n", response.Values[0].Datetime, response.Values[0].ADX)
}
```

## Error handling

The client returns `*APIError` when Twelve Data reports an API-level error,
including cases where the HTTP status is `200` but the JSON payload has
`"status": "error"`.

## Versioning and changelog

Use tagged releases for dependency pinning. Changelog entries are published in:
- `CHANGELOG.md`

## Local release process

This repository uses a local tag-based release flow (no release-please automation).

```bash
# 1) update CHANGELOG.md
git add CHANGELOG.md
git commit -m "chore: release v0.2.0"

# 2) create and push the tag
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin main
git push origin v0.2.0
```
