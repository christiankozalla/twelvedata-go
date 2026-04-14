# twelvedata-go

`twelvedata-go` is a read-only Go client for the Twelve Data REST API.

This module is the current base implementation of the Go port. The goal of this base is to provide a small, stable subset that is already useful in production and easy to extend with additional GET endpoints later.

## Status

This base client is intentionally limited to a supported subset of the Twelve Data API.

Supported now:
- Instrument catalogs: `StocksList`, `StockExchangesList`, `ForexPairsList`, `CryptocurrenciesList`, `ETFList`, `IndicesList`, `FundsList`, `BondsList`, `ExchangesList`, `TechnicalIndicatorsList`
- Discovery: `SymbolSearch`
- Market data: `ExchangeRate`, `CurrencyConversion`, `Quote`, `Price`, `EOD`
- Options: `OptionsExpiration`, `OptionsChain`
- Momentum indicators: `WILLR` (Williams %R), `ADX` (Average Directional Index), `PlusDI`, `MinusDI`
- Typed response structs: `PriceResponse`, `QuoteResponse`, `TimeSeriesResponse`, `WILLRResponse`, `ADXResponse`, `PlusDIResponse`, `MinusDIResponse`
- Builders with response helpers: `TimeSeries`, `Indicator`
- Shared request helpers: `AsURL`, `AsRawJSON`, `AsJSON`, `AsNormalized`, `AsNormalizedJSON`, `AsCSV`

Not implemented yet:
- Many fundamentals and corporate-data endpoints from the Python client, such as profile, dividends, splits, earnings, statistics, and financial statements
- Websocket support
- Strongly typed response structs for most endpoints

## Install

```bash
go get github.com/christiankozalla/twelvedata-go
```

## Use as a library in another Go app

Yes, this module is ready to be consumed as a library for the currently supported subset.

Practical notes:
- The client is read-only and only issues `GET` requests.
- Endpoint coverage is partial by design and expanding incrementally.
- API-level errors are returned as `*twelvedata.APIError`.

If you are developing both apps locally, you can use a `replace` directive in your consumer app's `go.mod`:

```go
module your-app

go 1.25

require github.com/christiankozalla/twelvedata-go v0.0.0

replace github.com/christiankozalla/twelvedata-go => ../path/to/twelvedata-go
```

## Create a client

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

    var response td.PriceResponse

    err := client.Price(td.PriceParams{Symbol: "AAPL"}).AsNormalized(ctx, &response)
    if err != nil {
        var apiErr *td.APIError
        if errors.As(err, &apiErr) {
            log.Fatalf("twelvedata api error: %s", apiErr.Error())
        }
        log.Fatal(err)
    }

    fmt.Printf("%s: %s\n", response.Symbol, response.Price)
}
```

## Quote example

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

## Time series example

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

## Indicator example

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

The client returns `APIError` when Twelve Data responds with an API-level error payload, including cases where the HTTP status is `200` but the JSON body contains `"status": "error"`.

## Extending the client

New endpoints should follow the existing pattern:
1. Add a params struct for the endpoint.
2. Map params into `url.Values` with the shared helper functions.
3. Return `c.newRequest("/endpoint", values)`.
4. Add typed response structs for the endpoint payload.
5. Add a higher-level helper only when the endpoint needs custom response shaping.

This keeps the transport and error handling centralized while making it straightforward to add more read-only endpoints from the Twelve Data documentation or the existing Python client.

## Recommended integration pattern

In larger applications, wrap the client behind a small interface so your business logic can be tested with mocks.

```go
package market

import (
    "context"

    td "github.com/christiankozalla/twelvedata-go"
)

type PriceProvider interface {
    LatestPrice(ctx context.Context, symbol string) (td.PriceResponse, error)
}

type TwelveDataPriceProvider struct {
    client *td.Client
}

func NewTwelveDataPriceProvider(client *td.Client) *TwelveDataPriceProvider {
    return &TwelveDataPriceProvider{client: client}
}

func (p *TwelveDataPriceProvider) LatestPrice(ctx context.Context, symbol string) (td.PriceResponse, error) {
    var out td.PriceResponse
    err := p.client.Price(td.PriceParams{Symbol: symbol}).AsNormalized(ctx, &out)
    return out, err
}
```

For tests in your app, provide a fake implementation of `PriceProvider` rather than calling the real API.

The same pattern works for typed indicators such as ADX:

```go
package market

import (
    "context"

    td "github.com/christiankozalla/twelvedata-go"
)

type ADXProvider interface {
    LatestADX(ctx context.Context, symbol, interval string, timePeriod int) (td.ADXResponse, error)
}

type TwelveDataADXProvider struct {
    client *td.Client
}

func NewTwelveDataADXProvider(client *td.Client) *TwelveDataADXProvider {
    return &TwelveDataADXProvider{client: client}
}

func (p *TwelveDataADXProvider) LatestADX(ctx context.Context, symbol, interval string, timePeriod int) (td.ADXResponse, error) {
    var out td.ADXResponse
    err := p.client.ADX(td.ADXParams{
        Symbol:     symbol,
        Interval:   interval,
        TimePeriod: &timePeriod,
        OutputSize: intPtr(1),
    }).AsJSON(ctx, &out)
    return out, err
}

func intPtr(v int) *int {
    value := v
    return &value
}
```

For tests in your app, provide a fake implementation of `ADXProvider` rather than calling the real API.

## CLI

A small manual test CLI is available under `cmd/tdcli`.
