package twelvedata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.twelvedata.com"
const requestSource = "go"

// Client wraps HTTP access to the Twelve Data REST API.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Option configures Client construction.
type Option func(*Client)

// NewClient builds a Client with sensible defaults.
func NewClient(apiKey string, opts ...Option) *Client {
	client := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}

	client.baseURL = strings.TrimRight(client.baseURL, "/")
	if client.baseURL == "" {
		client.baseURL = defaultBaseURL
	}

	return client
}

// WithBaseURL overrides the default REST endpoint.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithHTTPClient replaces the default HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// Request captures request metadata for subsequent execution.
type Request struct {
	client *Client
	path   string
	params url.Values
}

// APIError represents an error payload returned by the Twelve Data API.
type APIError struct {
	Code       int
	HTTPStatus int
	Message    string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}

	message := strings.TrimSpace(e.Message)
	switch {
	case e.Code != 0 && e.HTTPStatus != 0 && e.Code != e.HTTPStatus:
		return fmt.Sprintf("twelvedata: api error %d (http %d): %s", e.Code, e.HTTPStatus, message)
	case e.Code != 0:
		return fmt.Sprintf("twelvedata: api error %d: %s", e.Code, message)
	case e.HTTPStatus != 0:
		return fmt.Sprintf("twelvedata: http %d: %s", e.HTTPStatus, message)
	default:
		return fmt.Sprintf("twelvedata: %s", message)
	}
}

// Format controls the serialization of Twelve Data responses.
type Format string

const (
	// FormatJSON returns JSON payloads.
	FormatJSON Format = "JSON"
	// FormatCSV returns CSV payloads.
	FormatCSV Format = "CSV"
)

func (c *Client) newRequest(path string, params url.Values) *Request {
	clone := url.Values{}
	for key, values := range params {
		for _, value := range values {
			clone.Add(key, value)
		}
	}

	return &Request{
		client: c,
		path:   path,
		params: clone,
	}
}

func (r *Request) buildURL(format Format) (string, error) {
	if r.client == nil {
		return "", fmt.Errorf("twelvedata: request missing client")
	}

	base := r.client.baseURL
	if base == "" {
		base = defaultBaseURL
	}

	q := url.Values{}
	for key, values := range r.params {
		for _, value := range values {
			q.Add(key, value)
		}
	}

	if format == "" {
		format = FormatJSON
	}
	if r.client.apiKey == "" {
		return "", fmt.Errorf("twelvedata: missing API key")
	}

	q.Set("format", string(format))
	q.Set("apikey", r.client.apiKey)
	q.Set("source", requestSource)

	var sb strings.Builder
	sb.Grow(len(base) + len(r.path) + len(q.Encode()) + 2)
	sb.WriteString(base)
	sb.WriteString("/")
	sb.WriteString(strings.TrimPrefix(r.path, "/"))
	if encoded := q.Encode(); encoded != "" {
		sb.WriteString("?")
		sb.WriteString(encoded)
	}

	return sb.String(), nil
}

// AsURL returns the fully-qualified URL for the request using JSON format.
func (r *Request) AsURL() (string, error) {
	return r.buildURL(FormatJSON)
}

// AsRawJSON issues the HTTP request and returns the JSON payload bytes.
func (r *Request) AsRawJSON(ctx context.Context) ([]byte, error) {
	resp, err := r.do(ctx, FormatJSON)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// AsJSON issues the HTTP request and decodes the JSON response into out.
func (r *Request) AsJSON(ctx context.Context, out interface{}) error {
	payload, err := r.AsRawJSON(ctx)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, out)
}

// AsNormalized issues the HTTP request, applies standard Twelve Data normalization,
// and decodes the normalized payload into out.
func (r *Request) AsNormalized(ctx context.Context, out interface{}) error {
	payload, err := r.AsNormalizedJSON(ctx)
	if err != nil {
		return err
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, out)
}

// AsNormalizedJSON fetches the payload and applies Twelve Data's standard response normalization.
func (r *Request) AsNormalizedJSON(ctx context.Context) (interface{}, error) {
	rawPayload, err := r.AsRawJSON(ctx)
	if err != nil {
		return nil, err
	}

	var payload interface{}
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return nil, err
	}
	return normalizeJSON(payload), nil
}

// AsCSV issues the HTTP request and returns the CSV payload as a string.
func (r *Request) AsCSV(ctx context.Context) (string, error) {
	resp, err := r.do(ctx, FormatCSV)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (r *Request) do(ctx context.Context, format Format) (*http.Response, error) {
	fullURL, err := r.buildURL(format)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if format != FormatJSON {
		if resp.StatusCode >= http.StatusBadRequest {
			defer resp.Body.Close()
			payload, _ := io.ReadAll(resp.Body)
			return nil, buildAPIError(resp.StatusCode, payload)
		}
		return resp, nil
	}

	payload, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if err := apiErrorFromPayload(resp.StatusCode, payload); err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(payload))

	return resp, nil
}

type apiErrorPayload struct {
	Status  string `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func apiErrorFromPayload(httpStatus int, payload []byte) error {
	var apiPayload apiErrorPayload
	if err := json.Unmarshal(payload, &apiPayload); err == nil {
		if strings.EqualFold(apiPayload.Status, "error") {
			code := apiPayload.Code
			if code == 0 {
				code = httpStatus
			}
			return &APIError{
				Code:       code,
				HTTPStatus: httpStatus,
				Message:    firstNonEmpty(strings.TrimSpace(apiPayload.Message), compactPayload(payload)),
			}
		}
		if httpStatus >= http.StatusBadRequest {
			return &APIError{
				Code:       httpStatus,
				HTTPStatus: httpStatus,
				Message:    firstNonEmpty(strings.TrimSpace(apiPayload.Message), compactPayload(payload)),
			}
		}
		return nil
	}

	if httpStatus >= http.StatusBadRequest {
		return buildAPIError(httpStatus, payload)
	}
	return nil
}

func buildAPIError(httpStatus int, payload []byte) error {
	return &APIError{
		Code:       httpStatus,
		HTTPStatus: httpStatus,
		Message:    compactPayload(payload),
	}
}

func compactPayload(payload []byte) string {
	snippet := strings.TrimSpace(string(payload))
	if len(snippet) > 256 {
		snippet = snippet[:256]
	}
	return snippet
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// StocksListParams enumerates optional filters for the /stocks endpoint.
type StocksListParams struct {
	Symbol          string
	Exchange        string
	Country         string
	Type            string
	MICCode         string
	ShowPlan        *bool
	IncludeDelisted *bool
}

// StocksList returns metadata about stock instruments.
func (c *Client) StocksList(params StocksListParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "exchange", params.Exchange)
	addString(values, "country", params.Country)
	addString(values, "type", params.Type)
	addString(values, "mic_code", params.MICCode)
	addBool(values, "show_plan", params.ShowPlan)
	addBool(values, "include_delisted", params.IncludeDelisted)
	return c.newRequest("/stocks", values)
}

// StockExchangesList returns the /stock_exchanges catalog.
func (c *Client) StockExchangesList() *Request {
	return c.newRequest("/stock_exchanges", nil)
}

// ForexPairsListParams enumerates optional filters for the /forex_pairs endpoint.
type ForexPairsListParams struct {
	Symbol        string
	CurrencyBase  string
	CurrencyQuote string
}

// ForexPairsList returns the /forex_pairs catalog.
func (c *Client) ForexPairsList(params ForexPairsListParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "currency_base", params.CurrencyBase)
	addString(values, "currency_quote", params.CurrencyQuote)
	return c.newRequest("/forex_pairs", values)
}

// CryptocurrenciesListParams enumerates filters for /cryptocurrencies.
type CryptocurrenciesListParams struct {
	Symbol        string
	Exchange      string
	CurrencyBase  string
	CurrencyQuote string
}

// CryptocurrenciesList returns the /cryptocurrencies catalog.
func (c *Client) CryptocurrenciesList(params CryptocurrenciesListParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "exchange", params.Exchange)
	addString(values, "currency_base", params.CurrencyBase)
	addString(values, "currency_quote", params.CurrencyQuote)
	return c.newRequest("/cryptocurrencies", values)
}

// ETFListParams enumerates filters for /etf.
type ETFListParams struct {
	Symbol          string
	Exchange        string
	Country         string
	MICCode         string
	ShowPlan        *bool
	IncludeDelisted *bool
}

// ETFList returns the /etf catalog.
func (c *Client) ETFList(params ETFListParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "exchange", params.Exchange)
	addString(values, "country", params.Country)
	addString(values, "mic_code", params.MICCode)
	addBool(values, "show_plan", params.ShowPlan)
	addBool(values, "include_delisted", params.IncludeDelisted)
	return c.newRequest("/etf", values)
}

// IndicesListParams enumerates filters for /indices.
type IndicesListParams struct {
	Symbol          string
	Exchange        string
	Country         string
	MICCode         string
	ShowPlan        *bool
	IncludeDelisted *bool
}

// IndicesList returns the /indices catalog.
func (c *Client) IndicesList(params IndicesListParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "exchange", params.Exchange)
	addString(values, "country", params.Country)
	addString(values, "mic_code", params.MICCode)
	addBool(values, "show_plan", params.ShowPlan)
	addBool(values, "include_delisted", params.IncludeDelisted)
	return c.newRequest("/indices", values)
}

// FundsListParams enumerates filters for /funds.
type FundsListParams struct {
	Symbol          string
	Exchange        string
	Country         string
	Type            string
	ShowPlan        *bool
	IncludeDelisted *bool
	Page            *int
	OutputSize      *int
}

// FundsList returns the /funds catalog.
func (c *Client) FundsList(params FundsListParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "exchange", params.Exchange)
	addString(values, "country", params.Country)
	addString(values, "type", params.Type)
	addBool(values, "show_plan", params.ShowPlan)
	addBool(values, "include_delisted", params.IncludeDelisted)
	addInt(values, "page", params.Page)
	addInt(values, "outputsize", params.OutputSize)
	return c.newRequest("/funds", values)
}

// BondsListParams enumerates filters for /bonds.
type BondsListParams struct {
	Symbol          string
	Exchange        string
	Country         string
	Type            string
	ShowPlan        *bool
	IncludeDelisted *bool
	Page            *int
	OutputSize      *int
}

// BondsList returns the /bonds catalog.
func (c *Client) BondsList(params BondsListParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "exchange", params.Exchange)
	addString(values, "country", params.Country)
	addString(values, "type", params.Type)
	addBool(values, "show_plan", params.ShowPlan)
	addBool(values, "include_delisted", params.IncludeDelisted)
	addInt(values, "page", params.Page)
	addInt(values, "outputsize", params.OutputSize)
	return c.newRequest("/bonds", values)
}

// ExchangesListParams enumerates filters for /exchanges.
type ExchangesListParams struct {
	Name     string
	Code     string
	Country  string
	Type     string
	ShowPlan *bool
}

// ExchangesList returns the /exchanges catalog.
func (c *Client) ExchangesList(params ExchangesListParams) *Request {
	values := url.Values{}
	addString(values, "name", params.Name)
	addString(values, "code", params.Code)
	addString(values, "country", params.Country)
	addString(values, "type", params.Type)
	addBool(values, "show_plan", params.ShowPlan)
	return c.newRequest("/exchanges", values)
}

// TechnicalIndicatorsList returns supported indicator metadata.
func (c *Client) TechnicalIndicatorsList() *Request {
	return c.newRequest("/technical_indicators", nil)
}

// SymbolSearchParams enumerates filters for /symbol_search.
type SymbolSearchParams struct {
	Symbol     string
	OutputSize *int
	ShowPlan   *bool
}

// SymbolSearch returns the /symbol_search resource.
func (c *Client) SymbolSearch(params SymbolSearchParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addInt(values, "outputsize", params.OutputSize)
	addBool(values, "show_plan", params.ShowPlan)
	return c.newRequest("/symbol_search", values)
}

// LogoParams enumerates filters for /logo.
type LogoParams struct {
	Symbol   string
	Exchange string
	MICCode  string
	Country  string
}

// LogoResponse captures the /logo response.
type LogoResponse struct {
	Meta      LogoMeta `json:"meta"`
	URL       string   `json:"url,omitempty"`
	LogoBase  string   `json:"logo_base,omitempty"`
	LogoQuote string   `json:"logo_quote,omitempty"`
}

// LogoMeta captures /logo instrument metadata.
type LogoMeta struct {
	Symbol   string `json:"symbol,omitempty"`
	Exchange string `json:"exchange,omitempty"`
}

// Logo returns the /logo resource.
func (c *Client) Logo(params LogoParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "exchange", params.Exchange)
	addString(values, "mic_code", params.MICCode)
	addString(values, "country", params.Country)
	return c.newRequest("/logo", values)
}

// StatisticsParams enumerates filters for /statistics.
// At least one of Symbol, FIGI, ISIN, or CUSIP is expected by the API.
type StatisticsParams struct {
	Symbol   string
	FIGI     string
	ISIN     string
	CUSIP    string
	Exchange string
	MICCode  string
	Country  string
}

// StatisticsResponse captures the /statistics response.
type StatisticsResponse struct {
	Meta       StatisticsMeta `json:"meta"`
	Statistics StatisticsData `json:"statistics"`
}

// StatisticsMeta contains general instrument metadata for /statistics.
type StatisticsMeta struct {
	Symbol           string `json:"symbol,omitempty"`
	Name             string `json:"name,omitempty"`
	Currency         string `json:"currency,omitempty"`
	Exchange         string `json:"exchange,omitempty"`
	MICCode          string `json:"mic_code,omitempty"`
	ExchangeTimezone string `json:"exchange_timezone,omitempty"`
}

// StatisticsData contains all statistics sections returned by /statistics.
type StatisticsData struct {
	ValuationsMetrics  StatisticsValuationsMetrics  `json:"valuations_metrics"`
	Financials         StatisticsFinancials         `json:"financials"`
	StockStatistics    StatisticsStockStatistics    `json:"stock_statistics"`
	StockPriceSummary  StatisticsStockPriceSummary  `json:"stock_price_summary"`
	DividendsAndSplits StatisticsDividendsAndSplits `json:"dividends_and_splits"`
}

// StatisticsValuationsMetrics captures valuation metrics from /statistics.
type StatisticsValuationsMetrics struct {
	MarketCapitalization float64 `json:"market_capitalization,omitempty"`
	EnterpriseValue      float64 `json:"enterprise_value,omitempty"`
	TrailingPE           float64 `json:"trailing_pe,omitempty"`
	ForwardPE            float64 `json:"forward_pe,omitempty"`
	PEGRatio             float64 `json:"peg_ratio,omitempty"`
	PriceToSalesTTM      float64 `json:"price_to_sales_ttm,omitempty"`
	PriceToBookMRQ       float64 `json:"price_to_book_mrq,omitempty"`
	EnterpriseToRevenue  float64 `json:"enterprise_to_revenue,omitempty"`
	EnterpriseToEBITDA   float64 `json:"enterprise_to_ebitda,omitempty"`
}

// StatisticsFinancials captures financial metrics from /statistics.
type StatisticsFinancials struct {
	FiscalYearEnds    string                    `json:"fiscal_year_ends,omitempty"`
	MostRecentQuarter string                    `json:"most_recent_quarter,omitempty"`
	GrossMargin       float64                   `json:"gross_margin,omitempty"`
	ProfitMargin      float64                   `json:"profit_margin,omitempty"`
	OperatingMargin   float64                   `json:"operating_margin,omitempty"`
	ReturnOnAssetsTTM float64                   `json:"return_on_assets_ttm,omitempty"`
	ReturnOnEquityTTM float64                   `json:"return_on_equity_ttm,omitempty"`
	IncomeStatement   StatisticsIncomeStatement `json:"income_statement"`
	BalanceSheet      StatisticsBalanceSheet    `json:"balance_sheet"`
	CashFlow          StatisticsCashFlow        `json:"cash_flow"`
}

// StatisticsIncomeStatement captures income statement metrics from /statistics.
type StatisticsIncomeStatement struct {
	RevenueTTM                 float64 `json:"revenue_ttm,omitempty"`
	RevenuePerShareTTM         float64 `json:"revenue_per_share_ttm,omitempty"`
	QuarterlyRevenueGrowth     float64 `json:"quarterly_revenue_growth,omitempty"`
	GrossProfitTTM             float64 `json:"gross_profit_ttm,omitempty"`
	EBITDA                     float64 `json:"ebitda,omitempty"`
	NetIncomeToCommonTTM       float64 `json:"net_income_to_common_ttm,omitempty"`
	DilutedEPSTTM              float64 `json:"diluted_eps_ttm,omitempty"`
	QuarterlyEarningsGrowthYoY float64 `json:"quarterly_earnings_growth_yoy,omitempty"`
}

// StatisticsBalanceSheet captures balance sheet metrics from /statistics.
type StatisticsBalanceSheet struct {
	TotalCashMRQ         float64 `json:"total_cash_mrq,omitempty"`
	TotalCashPerShareMRQ float64 `json:"total_cash_per_share_mrq,omitempty"`
	TotalDebtMRQ         float64 `json:"total_debt_mrq,omitempty"`
	TotalDebtToEquityMRQ float64 `json:"total_debt_to_equity_mrq,omitempty"`
	CurrentRatioMRQ      float64 `json:"current_ratio_mrq,omitempty"`
	BookValuePerShareMRQ float64 `json:"book_value_per_share_mrq,omitempty"`
}

// StatisticsCashFlow captures cash flow metrics from /statistics.
type StatisticsCashFlow struct {
	OperatingCashFlowTTM   float64 `json:"operating_cash_flow_ttm,omitempty"`
	LeveredFreeCashFlowTTM float64 `json:"levered_free_cash_flow_ttm,omitempty"`
}

// StatisticsStockStatistics captures stock statistics from /statistics.
type StatisticsStockStatistics struct {
	SharesOutstanding               float64 `json:"shares_outstanding,omitempty"`
	FloatShares                     float64 `json:"float_shares,omitempty"`
	Avg10Volume                     float64 `json:"avg_10_volume,omitempty"`
	Avg90Volume                     float64 `json:"avg_90_volume,omitempty"`
	SharesShort                     float64 `json:"shares_short,omitempty"`
	ShortRatio                      float64 `json:"short_ratio,omitempty"`
	ShortPercentOfSharesOutstanding float64 `json:"short_percent_of_shares_outstanding,omitempty"`
	PercentHeldByInsiders           float64 `json:"percent_held_by_insiders,omitempty"`
	PercentHeldByInstitutions       float64 `json:"percent_held_by_institutions,omitempty"`
}

// StatisticsStockPriceSummary captures stock price summary fields from /statistics.
type StatisticsStockPriceSummary struct {
	FiftyTwoWeekLow    float64 `json:"fifty_two_week_low,omitempty"`
	FiftyTwoWeekHigh   float64 `json:"fifty_two_week_high,omitempty"`
	FiftyTwoWeekChange float64 `json:"fifty_two_week_change,omitempty"`
	Beta               float64 `json:"beta,omitempty"`
	Day50MA            float64 `json:"day_50_ma,omitempty"`
	Day200MA           float64 `json:"day_200_ma,omitempty"`
}

// StatisticsDividendsAndSplits captures dividends and split fields from /statistics.
type StatisticsDividendsAndSplits struct {
	ForwardAnnualDividendRate    float64 `json:"forward_annual_dividend_rate,omitempty"`
	ForwardAnnualDividendYield   float64 `json:"forward_annual_dividend_yield,omitempty"`
	TrailingAnnualDividendRate   float64 `json:"trailing_annual_dividend_rate,omitempty"`
	TrailingAnnualDividendYield  float64 `json:"trailing_annual_dividend_yield,omitempty"`
	FiveYearAverageDividendYield float64 `json:"5_year_average_dividend_yield,omitempty"`
	PayoutRatio                  float64 `json:"payout_ratio,omitempty"`
	DividendFrequency            string  `json:"dividend_frequency,omitempty"`
	DividendDate                 string  `json:"dividend_date,omitempty"`
	ExDividendDate               string  `json:"ex_dividend_date,omitempty"`
	LastSplitFactor              string  `json:"last_split_factor,omitempty"`
	LastSplitDate                string  `json:"last_split_date,omitempty"`
}

// Statistics returns the /statistics resource.
func (c *Client) Statistics(params StatisticsParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "figi", params.FIGI)
	addString(values, "isin", params.ISIN)
	addString(values, "cusip", params.CUSIP)
	addString(values, "exchange", params.Exchange)
	addString(values, "mic_code", params.MICCode)
	addString(values, "country", params.Country)
	return c.newRequest("/statistics", values)
}

// ExchangeRateParams enumerates filters for /exchange_rate.
type ExchangeRateParams struct {
	Symbol   string
	Date     string
	DP       *int
	Timezone string
}

// ExchangeRate returns the /exchange_rate resource.
func (c *Client) ExchangeRate(params ExchangeRateParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "date", params.Date)
	addInt(values, "dp", params.DP)
	addString(values, "timezone", params.Timezone)
	return c.newRequest("/exchange_rate", values)
}

// CurrencyConversionParams enumerates filters for /currency_conversion.
type CurrencyConversionParams struct {
	Symbol   string
	Amount   *float64
	Date     string
	DP       *int
	Timezone string
}

// CurrencyConversion returns the /currency_conversion resource.
func (c *Client) CurrencyConversion(params CurrencyConversionParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addFloat(values, "amount", params.Amount)
	addString(values, "date", params.Date)
	addInt(values, "dp", params.DP)
	addString(values, "timezone", params.Timezone)
	return c.newRequest("/currency_conversion", values)
}

// QuoteParams enumerates filters for /quote.
type QuoteParams struct {
	Symbol           string
	Interval         string
	Exchange         string
	Country          string
	VolumeTimePeriod string
	Type             string
	DP               *int
	Timezone         string
	Prepost          *bool
	MICCode          string
	EOD              string
	RollingPeriod    string
}

// QuoteResponse captures the normalized /quote response.
type QuoteResponse struct {
	Symbol         string            `json:"symbol,omitempty"`
	Name           string            `json:"name,omitempty"`
	Exchange       string            `json:"exchange,omitempty"`
	MicCode        string            `json:"mic_code,omitempty"`
	Currency       string            `json:"currency,omitempty"`
	Datetime       string            `json:"datetime,omitempty"`
	Timestamp      int64             `json:"timestamp,omitempty"`
	Open           string            `json:"open,omitempty"`
	High           string            `json:"high,omitempty"`
	Low            string            `json:"low,omitempty"`
	Close          string            `json:"close,omitempty"`
	Volume         string            `json:"volume,omitempty"`
	PreviousClose  string            `json:"previous_close,omitempty"`
	Change         string            `json:"change,omitempty"`
	PercentChange  string            `json:"percent_change,omitempty"`
	AverageVolume  string            `json:"average_volume,omitempty"`
	IsMarketOpen   bool              `json:"is_market_open,omitempty"`
	FiftyTwoWeek   QuoteFiftyTwoWeek `json:"fifty_two_week,omitempty"`
	RollingPeriod  string            `json:"rolling_period,omitempty"`
	VolumeInterval string            `json:"volume_time_period,omitempty"`
}

// QuoteFiftyTwoWeek captures the nested 52-week quote data.
type QuoteFiftyTwoWeek struct {
	Low               string `json:"low,omitempty"`
	High              string `json:"high,omitempty"`
	LowChange         string `json:"low_change,omitempty"`
	HighChange        string `json:"high_change,omitempty"`
	LowChangePercent  string `json:"low_change_percent,omitempty"`
	HighChangePercent string `json:"high_change_percent,omitempty"`
	Range             string `json:"range,omitempty"`
}

// Quote returns the /quote resource.
func (c *Client) Quote(params QuoteParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "interval", params.Interval)
	addString(values, "exchange", params.Exchange)
	addString(values, "country", params.Country)
	addString(values, "volume_time_period", params.VolumeTimePeriod)
	addString(values, "type", params.Type)
	addInt(values, "dp", params.DP)
	addString(values, "timezone", params.Timezone)
	addBool(values, "prepost", params.Prepost)
	addString(values, "mic_code", params.MICCode)
	addString(values, "eod", params.EOD)
	addString(values, "rolling_period", params.RollingPeriod)
	return c.newRequest("/quote", values)
}

// PriceParams enumerates filters for /price.
type PriceParams struct {
	Symbol   string
	Exchange string
	Country  string
	Type     string
	DP       *int
	Prepost  *bool
	MICCode  string
}

// PriceResponse captures the normalized /price response.
type PriceResponse struct {
	Symbol string `json:"symbol,omitempty"`
	Price  string `json:"price"`
}

// Price returns the /price resource.
func (c *Client) Price(params PriceParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "exchange", params.Exchange)
	addString(values, "country", params.Country)
	addString(values, "type", params.Type)
	addInt(values, "dp", params.DP)
	addBool(values, "prepost", params.Prepost)
	addString(values, "mic_code", params.MICCode)
	return c.newRequest("/price", values)
}

// EODParams enumerates filters for /eod.
type EODParams struct {
	Symbol   string
	Exchange string
	Country  string
	Type     string
	DP       *int
	Prepost  *bool
	MICCode  string
	Date     string
}

// EOD returns the /eod resource.
func (c *Client) EOD(params EODParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "exchange", params.Exchange)
	addString(values, "country", params.Country)
	addString(values, "type", params.Type)
	addInt(values, "dp", params.DP)
	addBool(values, "prepost", params.Prepost)
	addString(values, "mic_code", params.MICCode)
	addString(values, "date", params.Date)
	return c.newRequest("/eod", values)
}

// OptionsExpirationParams enumerates filters for /options/expiration.
type OptionsExpirationParams struct {
	Symbol   string
	Exchange string
	Country  string
	MICCode  string
}

// OptionsExpiration returns the /options/expiration resource.
func (c *Client) OptionsExpiration(params OptionsExpirationParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "exchange", params.Exchange)
	addString(values, "country", params.Country)
	addString(values, "mic_code", params.MICCode)
	return c.newRequest("/options/expiration", values)
}

// OptionsChainParams enumerates filters for /options/chain.
type OptionsChainParams struct {
	Symbol         string
	Exchange       string
	Country        string
	ExpirationDate string
	OptionID       string
	Side           string
	MICCode        string
}

// OptionsChain returns the /options/chain resource.
func (c *Client) OptionsChain(params OptionsChainParams) *Request {
	values := url.Values{}
	addString(values, "symbol", params.Symbol)
	addString(values, "exchange", params.Exchange)
	addString(values, "country", params.Country)
	addString(values, "expiration_date", params.ExpirationDate)
	addString(values, "option_id", params.OptionID)
	addString(values, "side", params.Side)
	addString(values, "mic_code", params.MICCode)
	return c.newRequest("/options/chain", values)
}

func addString(values url.Values, key, value string) {
	if values == nil {
		return
	}
	if value != "" {
		values.Set(key, value)
	}
}

func addBool(values url.Values, key string, value *bool) {
	if values == nil || value == nil {
		return
	}
	values.Set(key, strconv.FormatBool(*value))
}

func addInt(values url.Values, key string, value *int) {
	if values == nil || value == nil {
		return
	}
	values.Set(key, strconv.Itoa(*value))
}

func addFloat(values url.Values, key string, value *float64) {
	if values == nil || value == nil {
		return
	}
	values.Set(key, strconv.FormatFloat(*value, 'f', -1, 64))
}

func normalizeJSON(payload interface{}) interface{} {
	m, ok := payload.(map[string]interface{})
	if !ok {
		return payload
	}

	if status, ok := m["status"].(string); ok {
		if strings.EqualFold(status, "ok") {
			if result, ok := m["result"].(map[string]interface{}); ok {
				if list, ok := result["list"]; ok {
					return list
				}
			}
			if data, ok := m["data"]; ok {
				return data
			}
			if values, ok := m["values"]; ok {
				return values
			}
			if earnings, ok := m["earnings"]; ok {
				return earnings
			}
			return []interface{}{}
		}
		return payload
	}

	if data, ok := m["data"]; ok {
		return data
	}
	if values, ok := m["values"]; ok {
		return values
	}
	if earnings, ok := m["earnings"]; ok {
		return earnings
	}
	return payload
}
