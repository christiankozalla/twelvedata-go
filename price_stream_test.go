package twelvedata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

type capturedPriceStreamAction struct {
	Action string `json:"action"`
	Params *struct {
		Symbols json.RawMessage `json:"symbols"`
	} `json:"params,omitempty"`
}

func TestPriceStreamProtocolRoundTrip(t *testing.T) {
	type serverResult struct {
		apiKey  string
		actions []capturedPriceStreamAction
		err     error
	}
	result := make(chan serverResult, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			result <- serverResult{err: err}
			return
		}
		defer conn.CloseNow()

		captured := serverResult{apiKey: r.URL.Query().Get("apikey")}
		for range 4 {
			_, payload, readErr := conn.Read(context.Background())
			if readErr != nil {
				captured.err = readErr
				result <- captured
				return
			}

			var action capturedPriceStreamAction
			if decodeErr := json.Unmarshal(payload, &action); decodeErr != nil {
				captured.err = decodeErr
				result <- captured
				return
			}
			captured.actions = append(captured.actions, action)
		}

		messages := []string{
			`{
				"event": "subscribe-status",
				"status": "ok",
				"success": [{
					"symbol": "AAPL",
					"exchange": "NASDAQ",
					"country": "United States",
					"type": "Common Stock",
					"mic_code": "XNAS",
					"provider_id": "apple"
				}],
				"fails": [
					"UNKNOWN is not available",
					{"symbol": "BAD", "code": 404, "message": "not found", "retryable": false}
				],
				"request_id": "status-1"
			}`,
			`{
				"event": "price",
				"symbol": "AAPL",
				"currency": "USD",
				"exchange": "NASDAQ",
				"type": "Common Stock",
				"timestamp": "1592249566",
				"price": "342.0157",
				"day_volume": "27631112",
				"bid": 342.01,
				"ask": "342.02",
				"extended_hours": false
			}`,
			`{"event": "heartbeat-status", "sequence": 42}`,
		}
		for _, message := range messages {
			if writeErr := conn.Write(
				context.Background(),
				websocket.MessageText,
				[]byte(message),
			); writeErr != nil {
				captured.err = writeErr
				result <- captured
				return
			}
		}
		result <- captured
	}))
	defer server.Close()

	client := NewClient("stream-secret")
	stream, err := client.DialPriceStream(
		context.Background(),
		WithPriceStreamURL(server.URL),
	)
	if err != nil {
		t.Fatalf("DialPriceStream() error = %v", err)
	}
	defer stream.Close()

	if err := stream.Subscribe(context.Background(), []PriceStreamSymbol{
		{Symbol: "AAPL", Exchange: "NASDAQ"},
		{Symbol: "RY", MICCode: "XNYS"},
		{Symbol: "EUR/USD", Type: "Forex"},
	}); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if err := stream.Unsubscribe(context.Background(), []PriceStreamSymbol{
		{Symbol: "AAPL"},
		{Symbol: "MSFT"},
	}); err != nil {
		t.Fatalf("Unsubscribe() error = %v", err)
	}
	if err := stream.Reset(context.Background()); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if err := stream.Heartbeat(context.Background()); err != nil {
		t.Fatalf("Heartbeat() error = %v", err)
	}

	statusMessage, err := stream.Read(context.Background())
	if err != nil {
		t.Fatalf("Read(status) error = %v", err)
	}
	if statusMessage.Kind != PriceStreamMessageStatus {
		t.Fatalf("status Kind = %q, want %q", statusMessage.Kind, PriceStreamMessageStatus)
	}
	status := statusMessage.SubscriptionStatus
	if status == nil || status.Status != "ok" || len(status.Success) != 1 || len(status.Fails) != 2 {
		t.Fatalf("unexpected subscription status: %#v", status)
	}
	if status.Success[0].MICCode != "XNAS" ||
		string(status.Success[0].Extra["provider_id"]) != `"apple"` {
		t.Fatalf("unexpected successful instrument: %#v", status.Success[0])
	}
	if status.Fails[0].Message != "UNKNOWN is not available" {
		t.Fatalf("string failure message = %q", status.Fails[0].Message)
	}
	if string(status.Fails[1].Code) != "404" ||
		string(status.Fails[1].Extra["retryable"]) != "false" {
		t.Fatalf("unexpected structured failure: %#v", status.Fails[1])
	}
	if string(status.Extra["request_id"]) != `"status-1"` {
		t.Fatalf("status extra request_id = %s", status.Extra["request_id"])
	}

	priceMessage, err := stream.Read(context.Background())
	if err != nil {
		t.Fatalf("Read(price) error = %v", err)
	}
	price := priceMessage.Price
	if priceMessage.Kind != PriceStreamMessagePrice || price == nil {
		t.Fatalf("unexpected price message: %#v", priceMessage)
	}
	if price.Symbol != "AAPL" ||
		price.Timestamp != 1592249566 ||
		price.Price != 342.0157 ||
		price.DayVolume == nil ||
		*price.DayVolume != 27631112 ||
		price.Bid == nil ||
		*price.Bid != 342.01 ||
		price.Ask == nil ||
		*price.Ask != 342.02 {
		t.Fatalf("unexpected price event: %#v", price)
	}
	if string(price.Extra["extended_hours"]) != "false" {
		t.Fatalf("price extra extended_hours = %s", price.Extra["extended_hours"])
	}

	unknownMessage, err := stream.Read(context.Background())
	if err != nil {
		t.Fatalf("Read(unknown) error = %v", err)
	}
	if unknownMessage.Kind != PriceStreamMessageStatus ||
		unknownMessage.SubscriptionStatus == nil ||
		string(unknownMessage.SubscriptionStatus.Extra["sequence"]) != "42" {
		t.Fatalf("unexpected heartbeat status message: %#v", unknownMessage)
	}

	serverState := <-result
	if serverState.err != nil {
		t.Fatalf("fake server error = %v", serverState.err)
	}
	if serverState.apiKey != "stream-secret" {
		t.Fatalf("apikey query = %q", serverState.apiKey)
	}
	assertPriceStreamActions(t, serverState.actions)
}

func assertPriceStreamActions(t *testing.T, actions []capturedPriceStreamAction) {
	t.Helper()
	if len(actions) != 4 {
		t.Fatalf("received %d actions, want 4", len(actions))
	}

	if actions[0].Action != "subscribe" || actions[0].Params == nil {
		t.Fatalf("unexpected subscribe action: %#v", actions[0])
	}
	var extended []PriceStreamSymbol
	if err := json.Unmarshal(actions[0].Params.Symbols, &extended); err != nil {
		t.Fatalf("decode extended symbols: %v", err)
	}
	if len(extended) != 3 ||
		extended[0].Exchange != "NASDAQ" ||
		extended[1].MICCode != "XNYS" ||
		extended[2].Type != "Forex" {
		t.Fatalf("unexpected extended symbols: %#v", extended)
	}

	if actions[1].Action != "unsubscribe" || actions[1].Params == nil {
		t.Fatalf("unexpected unsubscribe action: %#v", actions[1])
	}
	var compact string
	if err := json.Unmarshal(actions[1].Params.Symbols, &compact); err != nil {
		t.Fatalf("decode compact symbols: %v", err)
	}
	if compact != "AAPL,MSFT" {
		t.Fatalf("compact symbols = %q", compact)
	}

	if actions[2].Action != "reset" || actions[2].Params != nil {
		t.Fatalf("unexpected reset action: %#v", actions[2])
	}
	if actions[3].Action != "heartbeat" || actions[3].Params != nil {
		t.Fatalf("unexpected heartbeat action: %#v", actions[3])
	}
}

func TestPriceStreamReadCancellationUnblocks(t *testing.T) {
	accepted := make(chan struct{})
	serverDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			close(serverDone)
			return
		}
		defer conn.CloseNow()
		close(accepted)
		_, _, _ = conn.Read(context.Background())
		close(serverDone)
	}))
	defer server.Close()

	stream, err := NewClient("secret").DialPriceStream(
		context.Background(),
		WithPriceStreamURL(server.URL),
	)
	if err != nil {
		t.Fatalf("DialPriceStream() error = %v", err)
	}
	defer stream.Close()
	<-accepted

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	startedAt := time.Now()
	_, err = stream.Read(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Read() error = %v, want context deadline exceeded", err)
	}
	var transportErr *PriceStreamTransportError
	if !errors.As(err, &transportErr) || transportErr.Operation != "read" {
		t.Fatalf("Read() error = %T %v, want read transport error", err, err)
	}
	if elapsed := time.Since(startedAt); elapsed > time.Second {
		t.Fatalf("Read() cancellation took %s", elapsed)
	}

	select {
	case <-serverDone:
	case <-time.After(time.Second):
		t.Fatal("fake server did not observe the cancelled connection")
	}
}

func TestPriceStreamReadReportsPeerClosure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		_ = conn.Close(websocket.StatusNormalClosure, "complete")
	}))
	defer server.Close()

	stream, err := NewClient("secret").DialPriceStream(
		context.Background(),
		WithPriceStreamURL(server.URL),
	)
	if err != nil {
		t.Fatalf("DialPriceStream() error = %v", err)
	}
	defer stream.Close()

	_, err = stream.Read(context.Background())
	var closeErr *PriceStreamClosedError
	if !errors.As(err, &closeErr) {
		t.Fatalf("Read() error = %T %v, want *PriceStreamClosedError", err, err)
	}
	if closeErr.Code != int(websocket.StatusNormalClosure) {
		t.Fatalf("close status = %d", closeErr.Code)
	}
}

func TestPriceStreamSupportsFullBudgetStatusPayload(t *testing.T) {
	success := make([]map[string]string, 500)
	for index := range success {
		success[index] = map[string]string{
			"symbol":       fmt.Sprintf("SYMBOL-%03d", index),
			"exchange":     "NASDAQ",
			"country":      "United States",
			"type":         "Common Stock",
			"display_name": strings.Repeat("Company ", 4),
		}
	}
	payload, err := json.Marshal(map[string]any{
		"event":   "subscribe-status",
		"status":  "ok",
		"success": success,
		"fails":   []any{},
	})
	if err != nil {
		t.Fatalf("encode status fixture: %v", err)
	}
	if len(payload) <= 32*1024 {
		t.Fatalf("fixture is only %d bytes; it must exceed the transport default", len(payload))
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, acceptErr := websocket.Accept(w, r, nil)
		if acceptErr != nil {
			return
		}
		defer conn.CloseNow()
		_ = conn.Write(context.Background(), websocket.MessageText, payload)
	}))
	defer server.Close()

	stream, err := NewClient("secret").DialPriceStream(
		context.Background(),
		WithPriceStreamURL(server.URL),
	)
	if err != nil {
		t.Fatalf("DialPriceStream() error = %v", err)
	}
	defer stream.Close()

	message, err := stream.Read(context.Background())
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if message.SubscriptionStatus == nil || len(message.SubscriptionStatus.Success) != 500 {
		t.Fatalf("decoded status = %#v", message.SubscriptionStatus)
	}
}

func TestPriceStreamDecodeErrors(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{name: "invalid JSON", payload: `{`},
		{name: "missing event", payload: `{"price": 1}`},
		{
			name:    "missing price",
			payload: `{"event":"price","symbol":"AAPL","timestamp":1592249566}`,
		},
		{
			name:    "non-finite price",
			payload: `{"event":"price","symbol":"AAPL","timestamp":1592249566,"price":"NaN"}`,
		},
		{
			name:    "fractional timestamp",
			payload: `{"event":"price","symbol":"AAPL","timestamp":1592249566.5,"price":1}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodePriceStreamMessage([]byte(test.payload))
			var decodeErr *PriceStreamDecodeError
			if !errors.As(err, &decodeErr) {
				t.Fatalf("error = %T %v, want *PriceStreamDecodeError", err, err)
			}
		})
	}
}

func TestPriceStreamUnknownEventPreservesRawPayload(t *testing.T) {
	payload := []byte(`{"event":"maintenance","until":123}`)
	message, err := decodePriceStreamMessage(payload)
	if err != nil {
		t.Fatalf("decodePriceStreamMessage() error = %v", err)
	}
	if message.Kind != PriceStreamMessageUnknown ||
		message.Event != "maintenance" ||
		string(message.Raw) != string(payload) {
		t.Fatalf("unexpected unknown message: %#v", message)
	}
}

func TestDialPriceStreamRedactsAPIKeyFromHandshakeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := NewClient("do-not-log-this-key").DialPriceStream(
		context.Background(),
		WithPriceStreamURL(server.URL),
	)
	var dialErr *PriceStreamDialError
	if !errors.As(err, &dialErr) {
		t.Fatalf("DialPriceStream() error = %T %v, want *PriceStreamDialError", err, err)
	}
	if dialErr.HTTPStatus != http.StatusUnauthorized {
		t.Fatalf("HTTPStatus = %d", dialErr.HTTPStatus)
	}
	if strings.Contains(err.Error(), "do-not-log-this-key") {
		t.Fatalf("error leaked API key: %v", err)
	}
}

func TestPriceStreamSymbolValidation(t *testing.T) {
	stream := &websocketPriceStream{conn: &recordingPriceStreamConnection{}}

	tests := []struct {
		name    string
		symbols []PriceStreamSymbol
	}{
		{name: "empty list"},
		{name: "empty symbol", symbols: []PriceStreamSymbol{{Symbol: " "}}},
		{name: "comma", symbols: []PriceStreamSymbol{{Symbol: "AAPL,MSFT"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := stream.Subscribe(context.Background(), test.symbols); err == nil {
				t.Fatal("Subscribe() error = nil")
			}
		})
	}
}

func TestPriceStreamSerializesConcurrentWrites(t *testing.T) {
	conn := &recordingPriceStreamConnection{writeDelay: time.Millisecond}
	stream := &websocketPriceStream{conn: conn}

	var wait sync.WaitGroup
	for range 50 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if err := stream.Heartbeat(context.Background()); err != nil {
				t.Errorf("Heartbeat() error = %v", err)
			}
		}()
	}
	wait.Wait()

	if got := conn.overlappingWrites.Load(); got != 0 {
		t.Fatalf("overlapping writes = %d", got)
	}
	if got := conn.writeCount.Load(); got != 50 {
		t.Fatalf("write count = %d, want 50", got)
	}
}

func TestPriceStreamCloseIsIdempotent(t *testing.T) {
	conn := &recordingPriceStreamConnection{}
	stream := &websocketPriceStream{conn: conn}

	if err := stream.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if got := conn.closeCount.Load(); got != 1 {
		t.Fatalf("close count = %d, want 1", got)
	}
}

func TestBuildPriceStreamURL(t *testing.T) {
	url, err := buildPriceStreamURL("wss://example.com/stream?region=eu&apikey=old", "new-key")
	if err != nil {
		t.Fatalf("buildPriceStreamURL() error = %v", err)
	}
	if url != "wss://example.com/stream?apikey=new-key&region=eu" {
		t.Fatalf("URL = %q", url)
	}

	for _, rawURL := range []string{"", "ftp://example.com/stream", "wss:///stream", "wss://example.com/#fragment"} {
		t.Run(fmt.Sprintf("reject %q", rawURL), func(t *testing.T) {
			if _, err := buildPriceStreamURL(rawURL, "secret"); err == nil {
				t.Fatal("buildPriceStreamURL() error = nil")
			}
		})
	}
}

type recordingPriceStreamConnection struct {
	activeWrites      atomic.Int32
	overlappingWrites atomic.Int32
	writeCount        atomic.Int32
	closeCount        atomic.Int32
	writeDelay        time.Duration
}

func (c *recordingPriceStreamConnection) Read(
	context.Context,
) (websocket.MessageType, []byte, error) {
	return 0, nil, errors.New("not implemented")
}

func (c *recordingPriceStreamConnection) Write(
	ctx context.Context,
	_ websocket.MessageType,
	_ []byte,
) error {
	if !c.activeWrites.CompareAndSwap(0, 1) {
		c.overlappingWrites.Add(1)
	}
	defer c.activeWrites.Store(0)

	if c.writeDelay > 0 {
		timer := time.NewTimer(c.writeDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	c.writeCount.Add(1)
	return nil
}

func (c *recordingPriceStreamConnection) Close(websocket.StatusCode, string) error {
	c.closeCount.Add(1)
	return nil
}
