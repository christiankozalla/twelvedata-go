package twelvedata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// PriceStreamMessageKind identifies a decoded realtime stream message.
type PriceStreamMessageKind string

const (
	// PriceStreamMessagePrice contains a realtime price tick.
	PriceStreamMessagePrice PriceStreamMessageKind = "price"
	// PriceStreamMessageStatus contains a subscription control acknowledgement.
	PriceStreamMessageStatus PriceStreamMessageKind = "status"
	// PriceStreamMessageUnknown contains a valid event that this package does not
	// interpret yet.
	PriceStreamMessageUnknown PriceStreamMessageKind = "unknown"
)

// PriceStreamSymbol identifies an instrument in a stream control request.
//
// A symbol with no disambiguating fields is encoded using Twelve Data's
// compact format. If any symbol in a request includes Exchange, MICCode, or
// Type, the complete request uses the extended object format.
type PriceStreamSymbol struct {
	Symbol   string `json:"symbol"`
	Exchange string `json:"exchange,omitempty"`
	MICCode  string `json:"mic_code,omitempty"`
	Type     string `json:"type,omitempty"`
}

// PriceEvent is a realtime price tick.
type PriceEvent struct {
	Event     string                     `json:"event"`
	Symbol    string                     `json:"symbol"`
	Currency  string                     `json:"currency,omitempty"`
	Exchange  string                     `json:"exchange,omitempty"`
	Type      string                     `json:"type,omitempty"`
	Timestamp int64                      `json:"timestamp"`
	Price     float64                    `json:"price"`
	DayVolume *float64                   `json:"day_volume,omitempty"`
	Bid       *float64                   `json:"bid,omitempty"`
	Ask       *float64                   `json:"ask,omitempty"`
	Extra     map[string]json.RawMessage `json:"-"`
}

// SubscriptionInstrument describes an instrument acknowledged by Twelve Data.
type SubscriptionInstrument struct {
	Symbol   string                     `json:"symbol"`
	Exchange string                     `json:"exchange,omitempty"`
	Country  string                     `json:"country,omitempty"`
	Type     string                     `json:"type,omitempty"`
	MICCode  string                     `json:"mic_code,omitempty"`
	Extra    map[string]json.RawMessage `json:"-"`
}

// SubscriptionFailure describes a rejected stream subscription. Code remains
// raw because Twelve Data may represent provider codes as numbers or strings.
type SubscriptionFailure struct {
	Symbol   string                     `json:"symbol,omitempty"`
	Exchange string                     `json:"exchange,omitempty"`
	Type     string                     `json:"type,omitempty"`
	MICCode  string                     `json:"mic_code,omitempty"`
	Message  string                     `json:"message,omitempty"`
	Code     json.RawMessage            `json:"code,omitempty"`
	Extra    map[string]json.RawMessage `json:"-"`
}

// SubscriptionStatus is the acknowledgement for subscribe, unsubscribe, or
// reset control requests.
type SubscriptionStatus struct {
	Event   string                     `json:"event"`
	Status  string                     `json:"status,omitempty"`
	Success []SubscriptionInstrument   `json:"success,omitempty"`
	Fails   []SubscriptionFailure      `json:"fails,omitempty"`
	Extra   map[string]json.RawMessage `json:"-"`
}

// PriceStreamMessage is one decoded event from the realtime stream.
type PriceStreamMessage struct {
	Kind               PriceStreamMessageKind
	Event              string
	Price              *PriceEvent
	SubscriptionStatus *SubscriptionStatus
	Raw                json.RawMessage
}

// PriceStreamDecodeError reports a syntactically invalid or semantically
// incomplete realtime event.
type PriceStreamDecodeError struct {
	Event string
	Err   error
}

func (e *PriceStreamDecodeError) Error() string {
	if e == nil {
		return ""
	}
	if e.Event == "" {
		return fmt.Sprintf("twelvedata: decode price stream message: %v", e.Err)
	}
	return fmt.Sprintf("twelvedata: decode price stream %q event: %v", e.Event, e.Err)
}

// Unwrap returns the underlying JSON or validation error.
func (e *PriceStreamDecodeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// PriceStreamClosedError reports a peer-initiated WebSocket close frame.
type PriceStreamClosedError struct {
	Code int
	Err  error
}

func (e *PriceStreamClosedError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("twelvedata: price stream closed with status %d", e.Code)
}

// Unwrap returns the transport's close error.
func (e *PriceStreamClosedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// PriceStreamTransportError reports a connected stream read or write failure
// that was not a peer close frame or message decoding failure.
type PriceStreamTransportError struct {
	Operation string
	Err       error
}

func (e *PriceStreamTransportError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("twelvedata: price stream %s failed: %v", e.Operation, e.Err)
}

// Unwrap returns the underlying transport or context error.
func (e *PriceStreamTransportError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// PriceStreamDialError reports a failed WebSocket handshake without including
// the API-key-bearing request URL in its error text.
type PriceStreamDialError struct {
	HTTPStatus int
	Err        error
}

func (e *PriceStreamDialError) Error() string {
	if e == nil {
		return ""
	}
	if e.HTTPStatus != 0 {
		return fmt.Sprintf("twelvedata: price stream handshake failed with HTTP status %d", e.HTTPStatus)
	}
	return "twelvedata: price stream handshake failed"
}

// Unwrap returns the underlying handshake error.
func (e *PriceStreamDialError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func decodePriceStreamMessage(payload []byte) (PriceStreamMessage, error) {
	var envelope struct {
		Event string `json:"event"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return PriceStreamMessage{}, &PriceStreamDecodeError{Err: err}
	}
	envelope.Event = strings.TrimSpace(envelope.Event)
	if envelope.Event == "" {
		return PriceStreamMessage{}, &PriceStreamDecodeError{
			Err: fmt.Errorf("missing event"),
		}
	}

	switch {
	case envelope.Event == "price":
		var event PriceEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return PriceStreamMessage{}, &PriceStreamDecodeError{
				Event: envelope.Event,
				Err:   err,
			}
		}
		return PriceStreamMessage{
			Kind:  PriceStreamMessagePrice,
			Event: envelope.Event,
			Price: &event,
		}, nil
	case strings.HasSuffix(envelope.Event, "-status") || envelope.Event == "status":
		var status SubscriptionStatus
		if err := json.Unmarshal(payload, &status); err != nil {
			return PriceStreamMessage{}, &PriceStreamDecodeError{
				Event: envelope.Event,
				Err:   err,
			}
		}
		return PriceStreamMessage{
			Kind:               PriceStreamMessageStatus,
			Event:              envelope.Event,
			SubscriptionStatus: &status,
		}, nil
	default:
		return PriceStreamMessage{
			Kind:  PriceStreamMessageUnknown,
			Event: envelope.Event,
			Raw:   append(json.RawMessage(nil), payload...),
		}, nil
	}
}

// UnmarshalJSON accepts both JSON numbers and numeric strings for provider
// numeric fields while validating the fields required to aggregate a tick.
func (e *PriceEvent) UnmarshalJSON(data []byte) error {
	*e = PriceEvent{}

	fields, err := decodeObject(data)
	if err != nil {
		return err
	}

	if err := decodeStringField(fields, "event", &e.Event); err != nil {
		return err
	}
	if err := decodeStringField(fields, "symbol", &e.Symbol); err != nil {
		return err
	}
	if strings.TrimSpace(e.Symbol) == "" {
		return fmt.Errorf("missing symbol")
	}
	if err := decodeStringField(fields, "currency", &e.Currency); err != nil {
		return err
	}
	if err := decodeStringField(fields, "exchange", &e.Exchange); err != nil {
		return err
	}
	if err := decodeStringField(fields, "type", &e.Type); err != nil {
		return err
	}

	timestamp, present, err := decodeInt64Field(fields, "timestamp")
	if err != nil {
		return err
	}
	if !present || timestamp <= 0 {
		return fmt.Errorf("missing or invalid timestamp")
	}
	e.Timestamp = timestamp

	price, present, err := decodeFloatField(fields, "price")
	if err != nil {
		return err
	}
	if !present {
		return fmt.Errorf("missing price")
	}
	e.Price = price

	if e.DayVolume, _, err = decodeOptionalFloatField(fields, "day_volume"); err != nil {
		return err
	}
	if e.Bid, _, err = decodeOptionalFloatField(fields, "bid"); err != nil {
		return err
	}
	if e.Ask, _, err = decodeOptionalFloatField(fields, "ask"); err != nil {
		return err
	}

	e.Extra = extraFields(fields,
		"event",
		"symbol",
		"currency",
		"exchange",
		"type",
		"timestamp",
		"price",
		"day_volume",
		"bid",
		"ask",
	)
	return nil
}

func (i *SubscriptionInstrument) UnmarshalJSON(data []byte) error {
	type instrumentAlias SubscriptionInstrument
	var decoded instrumentAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*i = SubscriptionInstrument(decoded)

	fields, err := decodeObject(data)
	if err != nil {
		return err
	}
	i.Extra = extraFields(fields, "symbol", "exchange", "country", "type", "mic_code")
	return nil
}

func (f *SubscriptionFailure) UnmarshalJSON(data []byte) error {
	*f = SubscriptionFailure{}

	if len(bytes.TrimSpace(data)) > 0 && bytes.TrimSpace(data)[0] == '"' {
		return json.Unmarshal(data, &f.Message)
	}

	type failureAlias SubscriptionFailure
	var decoded failureAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*f = SubscriptionFailure(decoded)

	fields, err := decodeObject(data)
	if err != nil {
		return err
	}
	f.Extra = extraFields(fields, "symbol", "exchange", "type", "mic_code", "message", "code")
	return nil
}

func (s *SubscriptionStatus) UnmarshalJSON(data []byte) error {
	type statusAlias SubscriptionStatus
	var decoded statusAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*s = SubscriptionStatus(decoded)

	fields, err := decodeObject(data)
	if err != nil {
		return err
	}
	s.Extra = extraFields(fields, "event", "status", "success", "fails")
	return nil
}

func decodeObject(data []byte) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, fmt.Errorf("expected JSON object")
	}
	return fields, nil
}

func decodeStringField(fields map[string]json.RawMessage, key string, target *string) error {
	raw, ok := fields[key]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	return nil
}

func decodeFloatField(fields map[string]json.RawMessage, key string) (float64, bool, error) {
	raw, ok := fields[key]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return 0, false, nil
	}

	token := strings.TrimSpace(string(raw))
	if strings.HasPrefix(token, `"`) {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return 0, false, fmt.Errorf("%s: %w", key, err)
		}
		token = strings.TrimSpace(value)
	}
	if token == "" {
		return 0, false, fmt.Errorf("%s: empty numeric value", key)
	}

	value, err := strconv.ParseFloat(token, 64)
	if err != nil {
		return 0, false, fmt.Errorf("%s: %w", key, err)
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false, fmt.Errorf("%s: non-finite numeric value", key)
	}
	return value, true, nil
}

func decodeOptionalFloatField(
	fields map[string]json.RawMessage,
	key string,
) (*float64, bool, error) {
	value, present, err := decodeFloatField(fields, key)
	if err != nil || !present {
		return nil, present, err
	}
	return &value, true, nil
}

func decodeInt64Field(fields map[string]json.RawMessage, key string) (int64, bool, error) {
	value, present, err := decodeFloatField(fields, key)
	if err != nil || !present {
		return 0, present, err
	}
	if math.Trunc(value) != value || value > math.MaxInt64 || value < math.MinInt64 {
		return 0, false, fmt.Errorf("%s: expected integer", key)
	}
	return int64(value), true, nil
}

func extraFields(
	fields map[string]json.RawMessage,
	known ...string,
) map[string]json.RawMessage {
	for _, key := range known {
		delete(fields, key)
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}
