package ws

import "encoding/json"

const (
	actionSubscribe   = "subscribe"
	actionUnsubscribe = "unsubscribe"
)

// inboundMessage is one client→server control frame.
type inboundMessage struct {
	Action  string   `json:"action"`
	Channel string   `json:"channel"`
	Symbols []string `json:"symbols"`
}

// errorEnvelope is the server→client reply for rejected input. Its Type
// mirrors the data envelopes so clients dispatch on one field; the connection
// always survives a rejected frame.
type errorEnvelope struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// marshalError renders an error envelope; the inputs are plain strings, so
// marshaling cannot fail and any payload is safe to deliver as-is.
func marshalError(message string) []byte {
	payload, err := json.Marshal(errorEnvelope{Type: "error", Message: message})
	if err != nil {
		return []byte(`{"type":"error","message":"internal error"}`)
	}
	return payload
}
