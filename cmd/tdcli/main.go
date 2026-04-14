package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	td "github.com/christiankozalla/twelvedata-go"
)

func main() {
	global := flag.NewFlagSet("twelvedata", flag.ExitOnError)
	apiKey := global.String("apikey", "", "Twelve Data API key (defaults to TWELVEDATA_API_KEY)")
	baseURL := global.String("base-url", "", "Override the Twelve Data API base URL")
	timeout := global.Duration("timeout", 30*time.Second, "HTTP timeout for API requests")

	if err := global.Parse(os.Args[1:]); err != nil {
		log.Fatalf("parse flags: %v", err)
	}

	args := global.Args()
	if len(args) == 0 {
		usage(global)
		os.Exit(2)
	}

	key := strings.TrimSpace(*apiKey)
	if key == "" {
		key = strings.TrimSpace(os.Getenv("TWELVEDATA_API_KEY"))
	}
	if key == "" {
		log.Fatalf("provide an API key with --apikey or TWELVEDATA_API_KEY")
	}

	httpClient := &http.Client{Timeout: *timeout}
	opts := []td.Option{td.WithHTTPClient(httpClient)}
	if trimmed := strings.TrimSpace(*baseURL); trimmed != "" {
		opts = append(opts, td.WithBaseURL(trimmed))
	}

	client := td.NewClient(key, opts...)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	switch args[0] {
	case "stocks":
		runStocks(ctx, client, args[1:])
	case "price":
		runPrice(ctx, client, args[1:])
	case "eod":
		runEOD(ctx, client, args[1:])
	case "time-series":
		runTimeSeries(ctx, client, args[1:])
	case "indicator":
		runIndicator(ctx, client, args[1:])
	default:
		log.Fatalf("unknown command %q", args[0])
	}
}

type stringMapFlag map[string]string

func (m stringMapFlag) String() string {
	if len(m) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, ",")
}

func (m stringMapFlag) Set(value string) error {
	if value == "" {
		return nil
	}
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid key=value pair: %s", value)
	}
	key := strings.TrimSpace(parts[0])
	if key == "" {
		return fmt.Errorf("empty key in pair: %s", value)
	}
	if _, exists := m[key]; exists {
		return fmt.Errorf("duplicate key %q", key)
	}
	m[key] = strings.TrimSpace(parts[1])
	return nil
}

func runStocks(ctx context.Context, client *td.Client, args []string) {
	fs := flag.NewFlagSet("stocks", flag.ExitOnError)
	symbol := fs.String("symbol", "", "Filter by symbol")
	exchange := fs.String("exchange", "", "Filter by exchange code")
	country := fs.String("country", "", "Filter by country code")
	typ := fs.String("type", "", "Filter by instrument type")
	includeDelisted := fs.Bool("include-delisted", false, "Include delisted instruments")
	mic := fs.String("mic", "", "Filter by MIC code")
	showPlan := fs.Bool("show-plan", false, "Include plan information in the response")

	if err := fs.Parse(args); err != nil {
		log.Fatalf("parse stocks flags: %v", err)
	}

	params := td.StocksListParams{
		Symbol:          *symbol,
		Exchange:        *exchange,
		Country:         *country,
		Type:            *typ,
		MICCode:         *mic,
		ShowPlan:        boolPtr(*showPlan),
		IncludeDelisted: boolPtr(*includeDelisted),
	}

	printJSON(ctx, client.StocksList(params))
}

func runPrice(ctx context.Context, client *td.Client, args []string) {
	fs := flag.NewFlagSet("price", flag.ExitOnError)
	symbol := fs.String("symbol", "", "Symbol to fetch the latest price for (required)")
	exchange := fs.String("exchange", "", "Optional exchange code")
	country := fs.String("country", "", "Optional country code")
	typ := fs.String("type", "", "Optional instrument type")
	dp := fs.Int("dp", 5, "Decimal places for numbers")
	prepost := fs.Bool("prepost", false, "Include pre/post market data")
	mic := fs.String("mic", "", "Optional MIC code")

	if err := fs.Parse(args); err != nil {
		log.Fatalf("parse price flags: %v", err)
	}

	if strings.TrimSpace(*symbol) == "" {
		fs.Usage()
		log.Fatalf("--symbol is required")
	}

	params := td.PriceParams{
		Symbol:   *symbol,
		Exchange: *exchange,
		Country:  *country,
		Type:     *typ,
		DP:       intPtr(*dp),
		Prepost:  boolPtr(*prepost),
		MICCode:  *mic,
	}

	printJSON(ctx, client.Price(params))
}

func runEOD(ctx context.Context, client *td.Client, args []string) {
	fs := flag.NewFlagSet("eod", flag.ExitOnError)
	symbol := fs.String("symbol", "", "Symbol to fetch end-of-day data for (required)")
	date := fs.String("date", "", "Retrieve a specific date (YYYY-MM-DD)")
	exchange := fs.String("exchange", "", "Optional exchange code")
	country := fs.String("country", "", "Optional country code")
	typ := fs.String("type", "", "Optional instrument type")
	dp := fs.Int("dp", 5, "Decimal places for numbers")
	prepost := fs.Bool("prepost", false, "Include pre/post market data")
	mic := fs.String("mic", "", "Optional MIC code")

	if err := fs.Parse(args); err != nil {
		log.Fatalf("parse eod flags: %v", err)
	}

	if strings.TrimSpace(*symbol) == "" {
		fs.Usage()
		log.Fatalf("--symbol is required")
	}

	params := td.EODParams{
		Symbol:   *symbol,
		Date:     *date,
		Exchange: *exchange,
		Country:  *country,
		Type:     *typ,
		DP:       intPtr(*dp),
		Prepost:  boolPtr(*prepost),
		MICCode:  *mic,
	}

	printJSON(ctx, client.EOD(params))
}

func runTimeSeries(ctx context.Context, client *td.Client, args []string) {
	fs := flag.NewFlagSet("time-series", flag.ExitOnError)
	symbol := fs.String("symbol", "", "Symbol to fetch (required)")
	interval := fs.String("interval", "1min", "Interval between datapoints")
	output := fs.Int("outputsize", 30, "Number of datapoints to include")
	start := fs.String("start-date", "", "Start date (YYYY-MM-DD)")
	end := fs.String("end-date", "", "End date (YYYY-MM-DD)")
	order := fs.String("order", "desc", "Sort order: asc or desc")

	if err := fs.Parse(args); err != nil {
		log.Fatalf("parse time-series flags: %v", err)
	}

	if strings.TrimSpace(*symbol) == "" {
		fs.Usage()
		log.Fatalf("--symbol is required")
	}

	params := td.TimeSeriesParams{
		Symbol:     *symbol,
		Interval:   *interval,
		OutputSize: intPtr(*output),
		StartDate:  *start,
		EndDate:    *end,
		Order:      *order,
	}

	series, err := client.TimeSeries(params).AsJSON(ctx)
	if err != nil {
		log.Fatalf("fetch time series: %v", err)
	}
	marshalAndPrint(series)
}

func runIndicator(ctx context.Context, client *td.Client, args []string) {
	fs := flag.NewFlagSet("indicator", flag.ExitOnError)
	name := fs.String("name", "", "Indicator endpoint name (e.g. rsi, adx) (required)")
	symbol := fs.String("symbol", "", "Symbol to request (required)")
	interval := fs.String("interval", "", "Optional interval")
	exchange := fs.String("exchange", "", "Optional exchange code")
	country := fs.String("country", "", "Optional country code")
	typ := fs.String("type", "", "Optional instrument type")
	seriesType := fs.String("series-type", "", "Series type (open/high/low/close)")
	timePeriod := fs.Int("time-period", 0, "Time period for the indicator")
	outputSize := fs.Int("outputsize", 0, "Number of datapoints to include")
	startDate := fs.String("start-date", "", "Start date (YYYY-MM-DD)")
	endDate := fs.String("end-date", "", "End date (YYYY-MM-DD)")
	dp := fs.Int("dp", 0, "Decimal places for numeric output")
	timezone := fs.String("timezone", "", "Timezone for the response")
	order := fs.String("order", "", "Sort order: asc or desc")
	prepost := fs.Bool("prepost", false, "Include pre/post market data")
	mic := fs.String("mic", "", "Optional MIC code")
	extra := make(stringMapFlag)
	fs.Var(extra, "param", "Additional key=value pairs for the request (repeatable)")

	if err := fs.Parse(args); err != nil {
		log.Fatalf("parse indicator flags: %v", err)
	}

	setFlags := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*symbol) == "" {
		fs.Usage()
		log.Fatalf("--name and --symbol are required")
	}

	indicatorParams := td.IndicatorParams{
		Symbol:     *symbol,
		Interval:   *interval,
		Exchange:   *exchange,
		Country:    *country,
		Type:       *typ,
		SeriesType: *seriesType,
		StartDate:  *startDate,
		EndDate:    *endDate,
		Timezone:   *timezone,
		Order:      *order,
		MICCode:    *mic,
	}
	if setFlags["time-period"] {
		indicatorParams.TimePeriod = intPtr(*timePeriod)
	}
	if setFlags["outputsize"] {
		indicatorParams.OutputSize = intPtr(*outputSize)
	}
	if setFlags["dp"] {
		indicatorParams.DP = intPtr(*dp)
	}
	if setFlags["prepost"] {
		indicatorParams.Prepost = boolPtr(*prepost)
	}
	if len(extra) > 0 {
		extras := make(map[string]string, len(extra))
		for k, v := range extra {
			extras[k] = v
		}
		indicatorParams.Extra = extras
	}

	printJSON(ctx, client.Indicator(*name, indicatorParams))
}

func printJSON(ctx context.Context, req *td.Request) {
	payload, err := req.AsNormalizedJSON(ctx)
	if err != nil {
		log.Fatalf("api request failed: %v", err)
	}
	marshalAndPrint(payload)
}

func marshalAndPrint(payload interface{}) {
	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Fatalf("encode response: %v", err)
	}
	fmt.Println(string(bytes))
}

func usage(fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, "Usage: %s [global options] <command> [command options]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nAvailable commands:\n")
	fmt.Fprintf(os.Stderr, "  stocks        List tradable instruments\n")
	fmt.Fprintf(os.Stderr, "  price         Fetch the latest price for a symbol\n")
	fmt.Fprintf(os.Stderr, "  eod           Fetch end-of-day pricing\n")
	fmt.Fprintf(os.Stderr, "  time-series   Fetch raw time series data\n")
	fmt.Fprintf(os.Stderr, "  indicator     Call an indicator endpoint\n")
	fmt.Fprintf(os.Stderr, "\nGlobal options:\n")
	fs.PrintDefaults()
}

func boolPtr(v bool) *bool {
	value := v
	return &value
}

func intPtr(v int) *int {
	value := v
	return &value
}
