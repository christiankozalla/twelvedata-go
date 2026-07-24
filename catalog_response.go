package twelvedata

// InstrumentListItem captures the common instrument metadata returned by the
// /stocks and /etf catalog endpoints.
type InstrumentListItem struct {
	Symbol   string `json:"symbol,omitempty"`
	Name     string `json:"name,omitempty"`
	Currency string `json:"currency,omitempty"`
	Exchange string `json:"exchange,omitempty"`
	MICCode  string `json:"mic_code,omitempty"`
	Country  string `json:"country,omitempty"`
	Type     string `json:"type,omitempty"`
	FIGICode string `json:"figi_code,omitempty"`
	CFICode  string `json:"cfi_code,omitempty"`
}

// StocksListResponse captures the /stocks catalog response.
type StocksListResponse struct {
	Status string               `json:"status,omitempty"`
	Count  int                  `json:"count,omitempty"`
	Data   []InstrumentListItem `json:"data"`
}

// ETFListResponse captures the /etf catalog response.
type ETFListResponse struct {
	Status string               `json:"status,omitempty"`
	Count  int                  `json:"count,omitempty"`
	Data   []InstrumentListItem `json:"data"`
}

// SymbolSearchResponse captures the /symbol_search response.
type SymbolSearchResponse struct {
	Status string             `json:"status,omitempty"`
	Data   []SymbolSearchItem `json:"data"`
}

// SymbolSearchItem captures one instrument returned by /symbol_search.
type SymbolSearchItem struct {
	Symbol         string `json:"symbol,omitempty"`
	InstrumentName string `json:"instrument_name,omitempty"`
	Exchange       string `json:"exchange,omitempty"`
	MICCode        string `json:"mic_code,omitempty"`
	InstrumentType string `json:"instrument_type,omitempty"`
	Country        string `json:"country,omitempty"`
	Currency       string `json:"currency,omitempty"`
}
