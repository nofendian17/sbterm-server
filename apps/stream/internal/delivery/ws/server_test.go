package ws

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/stream/internal/service"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

// wireEnvelope mirrors only the fields the tests dispatch on.
type wireEnvelope struct {
	Type    string `json:"type"`
	Symbol  string `json:"symbol"`
	Message string `json:"message"`
}

// frameReader owns the connection's single reader goroutine: gorilla forbids
// further reads once one fails, so every test consumes through here instead.
type frameReader struct {
	frames chan []byte
}

func startReader(conn *websocket.Conn) *frameReader {
	fr := &frameReader{frames: make(chan []byte, 128)}
	go func() {
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				close(fr.frames)
				return
			}
			select {
			case fr.frames <- raw:
			default: // drop overflow; tests never need more than the buffer holds
			}
		}
	}()
	return fr
}

// window collects every frame delivered within d.
func (fr *frameReader) window(d time.Duration) [][]byte {
	var out [][]byte
	deadline := time.Now().Add(d)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return out
		}
		select {
		case raw, ok := <-fr.frames:
			if !ok {
				return out
			}
			out = append(out, raw)
		case <-time.After(remaining):
			return out
		}
	}
}

// pumpBroadcasts repeatedly runs broadcasts and collects frames until pred
// holds over what was collected or the deadline passes.
func (fr *frameReader) pumpBroadcasts(broadcast func(), pred func([][]byte) bool, timeout time.Duration) [][]byte {
	var out [][]byte
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		broadcast()
		out = append(out, fr.window(40*time.Millisecond)...)
		if pred(out) {
			return out
		}
	}
	return out
}

func newTestServer(t *testing.T) (*httptest.Server, *service.Hub, *websocket.Conn, *frameReader) {
	t.Helper()
	hub := service.NewHub(log.New(log.WithWriter(io.Discard)))
	srv := httptest.NewServer(NewServer(hub).Router())
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	t.Cleanup(func() { _ = conn.Close() })
	return srv, hub, conn, startReader(conn)
}

func send(t *testing.T, conn *websocket.Conn, payload string) {
	t.Helper()
	require.NoError(t, conn.SetWriteDeadline(time.Now().Add(2*time.Second)))
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(payload)))
}

func decode(t *testing.T, raw []byte) wireEnvelope {
	t.Helper()
	var env wireEnvelope
	require.NoError(t, json.Unmarshal(raw, &env))
	return env
}

// envelope builds the JSON the poll loop would produce for one batch,
// mirroring service.runningTradeEnvelope.
func envelope(symbol string) []byte {
	raw, _ := json.Marshal(map[string]any{
		"type":   "running_trade",
		"symbol": symbol,
		"data":   []any{map[string]any{"price": 8250}},
	})
	return raw
}

func TestSubscribeFiltersBySymbol(t *testing.T) {
	_, hub, conn, fr := newTestServer(t)

	send(t, conn, `{"action":"subscribe","channel":"running_trade","symbols":["BBCA"]}`)

	broadcast := func() {
		hub.Broadcast(service.ChannelRunningTrade, "BBCA", envelope("BBCA"))
		hub.Broadcast(service.ChannelRunningTrade, "ANTM", envelope("ANTM"))
	}
	frames := fr.pumpBroadcasts(broadcast, func(got [][]byte) bool { return len(got) > 0 }, 3*time.Second)

	require.NotEmpty(t, frames, "expected at least one delivered frame")
	for _, raw := range frames {
		env := decode(t, raw)
		assert.Equal(t, "running_trade", env.Type)
		assert.Equal(t, "BBCA", env.Symbol, "only subscribed symbols may pass")
	}
}

func TestNoDataWithoutSubscription(t *testing.T) {
	_, hub, _, fr := newTestServer(t)

	deadline := time.Now().Add(400 * time.Millisecond)
	for time.Now().Before(deadline) {
		hub.Broadcast(service.ChannelRunningTrade, "BBCA", []byte(`{"k":1}`))
		time.Sleep(50 * time.Millisecond)
	}

	assert.Empty(t, fr.window(200*time.Millisecond), "unsubscribed clients receive nothing")
}

func TestUnknownChannelRepliesErrorAndConnectionSurvives(t *testing.T) {
	_, hub, conn, fr := newTestServer(t)

	send(t, conn, `{"action":"subscribe","channel":"liveprice","symbols":["BBCA"]}`)
	frames := fr.window(2 * time.Second)
	require.NotEmpty(t, frames)
	assertErrorEnvelope(t, decode(t, frames[0]), "unknown channel")

	send(t, conn, `{"action":"subscribe","channel":"running_trade","symbols":["BBCA"]}`)
	got := fr.pumpBroadcasts(
		func() { hub.Broadcast(service.ChannelRunningTrade, "BBCA", []byte(`{"k":1}`)) },
		func(f [][]byte) bool { return len(f) > 0 },
		3*time.Second,
	)
	assert.NotEmpty(t, got, "connection should deliver data after a rejected frame")
}

func TestMalformedJSONRepliesError(t *testing.T) {
	_, _, conn, fr := newTestServer(t)

	send(t, conn, `this is not json`)
	frames := fr.window(2 * time.Second)
	require.NotEmpty(t, frames)
	assertErrorEnvelope(t, decode(t, frames[0]), "invalid message")
}

func TestUnknownActionRepliesError(t *testing.T) {
	_, _, conn, fr := newTestServer(t)

	send(t, conn, `{"action":"dance","channel":"running_trade"}`)
	frames := fr.window(2 * time.Second)
	require.NotEmpty(t, frames)
	assertErrorEnvelope(t, decode(t, frames[0]), "unknown action")
}

func TestBroadcastModeWithEmptySymbols(t *testing.T) {
	_, hub, conn, fr := newTestServer(t)

	send(t, conn, `{"action":"subscribe","channel":"running_trade","symbols":[]}`)

	seen := map[string]bool{}
	frames := fr.pumpBroadcasts(
		func() {
			hub.Broadcast(service.ChannelRunningTrade, "BBCA", envelope("BBCA"))
			hub.Broadcast(service.ChannelRunningTrade, "ANTM", envelope("ANTM"))
		},
		func(f [][]byte) bool {
			for _, raw := range f {
				seen[decode(t, raw).Symbol] = true
			}
			return seen["BBCA"] && seen["ANTM"]
		},
		3*time.Second,
	)
	assert.True(t, seen["BBCA"] && seen["ANTM"], "broadcast mode receives every symbol, saw %v", seen)
	_ = frames
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	_, hub, conn, fr := newTestServer(t)

	send(t, conn, `{"action":"subscribe","channel":"running_trade","symbols":["BBCA"]}`)
	broadcast := func() { hub.Broadcast(service.ChannelRunningTrade, "BBCA", []byte(`{"k":1}`)) }

	before := fr.pumpBroadcasts(broadcast, func(got [][]byte) bool { return len(got) > 0 }, 3*time.Second)
	require.NotEmpty(t, before)

	send(t, conn, `{"action":"unsubscribe","channel":"running_trade","symbols":["BBCA"]}`)
	time.Sleep(100 * time.Millisecond) // let the read pump apply it

	for i := 0; i < 5; i++ {
		broadcast()
		time.Sleep(20 * time.Millisecond)
	}
	assert.Empty(t, fr.window(300*time.Millisecond), "no delivery after unsubscribe")
}

func assertErrorEnvelope(t *testing.T, env wireEnvelope, wantIn string) {
	t.Helper()
	assert.Equal(t, "error", env.Type)
	assert.Contains(t, env.Message, wantIn)
}

func TestHealthz(t *testing.T) {
	srv, _, _, _ := newTestServer(t)

	resp, err := http.Get(srv.URL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}
