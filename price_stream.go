package twelvedata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/coder/websocket"
)

const (
	defaultPriceStreamURL       = "wss://ws.twelvedata.com/v1/quotes/price"
	defaultPriceStreamReadLimit = 1 << 20
)

// PriceStreamOption configures a realtime price stream connection.
type PriceStreamOption interface {
	applyPriceStream(*priceStreamConfig)
}

type priceStreamOptionFunc func(*priceStreamConfig)

func (option priceStreamOptionFunc) applyPriceStream(config *priceStreamConfig) {
	option(config)
}

type priceStreamConfig struct {
	url string
}

// WithPriceStreamURL overrides the default realtime endpoint. It is intended
// for deterministic integration tests and compatible proxy endpoints.
func WithPriceStreamURL(rawURL string) PriceStreamOption {
	return priceStreamOptionFunc(func(config *priceStreamConfig) {
		config.url = strings.TrimSpace(rawURL)
	})
}

// PriceStream is a connected Twelve Data realtime price stream.
//
// Callers must use one goroutine for Read. Control methods are safe for
// concurrent use and are serialized. Cancelling an active Read may make the
// underlying connection unusable; callers should reconnect.
type PriceStream interface {
	Subscribe(ctx context.Context, symbols []PriceStreamSymbol) error
	Unsubscribe(ctx context.Context, symbols []PriceStreamSymbol) error
	Reset(ctx context.Context) error
	Heartbeat(ctx context.Context) error
	Read(ctx context.Context) (PriceStreamMessage, error)
	Close() error
}

type priceStreamConnection interface {
	Read(context.Context) (websocket.MessageType, []byte, error)
	Write(context.Context, websocket.MessageType, []byte) error
	Close(websocket.StatusCode, string) error
}

type websocketPriceStream struct {
	conn      priceStreamConnection
	writeMu   sync.Mutex
	closeOnce sync.Once
	closeErr  error
}

type priceStreamAction struct {
	Action string                   `json:"action"`
	Params *priceStreamActionParams `json:"params,omitempty"`
}

type priceStreamActionParams struct {
	Symbols priceStreamSymbols `json:"symbols"`
}

type priceStreamSymbols []PriceStreamSymbol

// DialPriceStream connects to Twelve Data's realtime price endpoint using the
// API key and HTTP transport configured on Client.
func (c *Client) DialPriceStream(
	ctx context.Context,
	opts ...PriceStreamOption,
) (PriceStream, error) {
	if c == nil {
		return nil, fmt.Errorf("twelvedata: price stream missing client")
	}
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, fmt.Errorf("twelvedata: missing API key")
	}

	config := priceStreamConfig{url: defaultPriceStreamURL}
	for _, opt := range opts {
		if opt != nil {
			opt.applyPriceStream(&config)
		}
	}

	endpoint, err := buildPriceStreamURL(config.url, c.apiKey)
	if err != nil {
		return nil, err
	}

	conn, response, err := websocket.Dial(ctx, endpoint, &websocket.DialOptions{
		HTTPClient: c.httpClient,
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		return nil, &PriceStreamDialError{
			HTTPStatus: status,
			Err:        err,
		}
	}
	conn.SetReadLimit(defaultPriceStreamReadLimit)

	return &websocketPriceStream{conn: conn}, nil
}

func buildPriceStreamURL(rawURL string, apiKey string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("twelvedata: missing price stream URL")
	}

	endpoint, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("twelvedata: invalid price stream URL: %w", err)
	}
	switch endpoint.Scheme {
	case "ws", "wss", "http", "https":
	default:
		return "", fmt.Errorf("twelvedata: unsupported price stream URL scheme %q", endpoint.Scheme)
	}
	if endpoint.Host == "" {
		return "", fmt.Errorf("twelvedata: price stream URL missing host")
	}
	if endpoint.Fragment != "" {
		return "", fmt.Errorf("twelvedata: price stream URL must not contain a fragment")
	}

	query := endpoint.Query()
	query.Set("apikey", apiKey)
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
}

func (s *websocketPriceStream) Subscribe(
	ctx context.Context,
	symbols []PriceStreamSymbol,
) error {
	normalized, err := normalizePriceStreamSymbols(symbols)
	if err != nil {
		return err
	}
	return s.writeAction(ctx, priceStreamAction{
		Action: "subscribe",
		Params: &priceStreamActionParams{Symbols: normalized},
	})
}

func (s *websocketPriceStream) Unsubscribe(
	ctx context.Context,
	symbols []PriceStreamSymbol,
) error {
	normalized, err := normalizePriceStreamSymbols(symbols)
	if err != nil {
		return err
	}
	return s.writeAction(ctx, priceStreamAction{
		Action: "unsubscribe",
		Params: &priceStreamActionParams{Symbols: normalized},
	})
}

func (s *websocketPriceStream) Reset(ctx context.Context) error {
	return s.writeAction(ctx, priceStreamAction{Action: "reset"})
}

func (s *websocketPriceStream) Heartbeat(ctx context.Context) error {
	return s.writeAction(ctx, priceStreamAction{Action: "heartbeat"})
}

func (s *websocketPriceStream) writeAction(ctx context.Context, action priceStreamAction) error {
	if s == nil || s.conn == nil {
		return fmt.Errorf("twelvedata: price stream is not connected")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("twelvedata: encode price stream %s action: %w", action.Action, err)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.conn.Write(ctx, websocket.MessageText, payload); err != nil {
		return &PriceStreamTransportError{
			Operation: "write " + action.Action,
			Err:       err,
		}
	}
	return nil
}

func (s *websocketPriceStream) Read(ctx context.Context) (PriceStreamMessage, error) {
	if s == nil || s.conn == nil {
		return PriceStreamMessage{}, fmt.Errorf("twelvedata: price stream is not connected")
	}

	_, payload, err := s.conn.Read(ctx)
	if err != nil {
		if status := websocket.CloseStatus(err); status != -1 {
			return PriceStreamMessage{}, &PriceStreamClosedError{
				Code: int(status),
				Err:  err,
			}
		}
		return PriceStreamMessage{}, &PriceStreamTransportError{
			Operation: "read",
			Err:       err,
		}
	}

	return decodePriceStreamMessage(payload)
}

func (s *websocketPriceStream) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		if err := s.conn.Close(websocket.StatusNormalClosure, ""); err != nil {
			s.closeErr = &PriceStreamTransportError{
				Operation: "close",
				Err:       err,
			}
		}
	})
	return s.closeErr
}

func normalizePriceStreamSymbols(symbols []PriceStreamSymbol) (priceStreamSymbols, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("twelvedata: price stream symbols must not be empty")
	}

	normalized := make(priceStreamSymbols, len(symbols))
	for index, symbol := range symbols {
		symbol.Symbol = strings.TrimSpace(symbol.Symbol)
		symbol.Exchange = strings.TrimSpace(symbol.Exchange)
		symbol.MICCode = strings.TrimSpace(symbol.MICCode)
		symbol.Type = strings.TrimSpace(symbol.Type)
		if symbol.Symbol == "" {
			return nil, fmt.Errorf("twelvedata: price stream symbol at index %d is empty", index)
		}
		if strings.Contains(symbol.Symbol, ",") {
			return nil, fmt.Errorf(
				"twelvedata: price stream symbol %q must not contain a comma",
				symbol.Symbol,
			)
		}
		normalized[index] = symbol
	}
	return normalized, nil
}

func (symbols priceStreamSymbols) MarshalJSON() ([]byte, error) {
	extended := false
	compact := make([]string, len(symbols))
	for index, symbol := range symbols {
		compact[index] = symbol.Symbol
		if symbol.Exchange != "" || symbol.MICCode != "" || symbol.Type != "" {
			extended = true
		}
	}
	if !extended {
		return json.Marshal(strings.Join(compact, ","))
	}
	return json.Marshal([]PriceStreamSymbol(symbols))
}
