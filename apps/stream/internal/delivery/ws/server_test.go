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
	// Converge: while the subscribe is still in flight the client is in
	// default broadcast and ANTM passes; once applied, only BBCA does.
	frames := fr.pumpBroadcasts(broadcast, func(got [][]byte) bool {
		sawANTM := false
		sawBBCAAfter := false
		for _, raw := range got {
			sym := decode(t, raw).Symbol
			if sym == "ANTM" {
				sawANTM = true
			}
			if sym == "BBCA" && sawANTM {
				sawBBCAAfter = true
			}
		}
		return sawANTM && sawBBCAAfter
	}, 3*time.Second)

	require.NotEmpty(t, frames, "expected at least one delivered frame")
	// Only frames after the last ANTM are guaranteed post-subscribe.
	lastANTM := -1
	for i, raw := range frames {
		if decode(t, raw).Symbol == "ANTM" {
			lastANTM = i
		}
	}
	for _, raw := range frames[lastANTM+1:] {
		env := decode(t, raw)
		assert.Equal(t, "running_trade", env.Type)
		assert.Equal(t, "BBCA", env.Symbol, "only subscribed symbols may pass after filter applied")
	}
}

func TestUnsubscribedClientReceivesEverything(t *testing.T) {
	// Spec: "Broadcast semua batch secara default".
	_, hub, _, fr := newTestServer(t)

	frames := fr.pumpBroadcasts(
		func() { hub.Broadcast(service.ChannelRunningTrade, "BBCA", []byte(`{"k":1}`)) },
		func(f [][]byte) bool { return len(f) > 0 },
		3*time.Second,
	)
	assert.NotEmpty(t, frames, "clients that never subscribed receive all batches")
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

func TestUnsubscribeShrinksFilter(t *testing.T) {
	_, hub, conn, fr := newTestServer(t)

	send(t, conn, `{"action":"subscribe","channel":"running_trade","symbols":["BBCA","ANTM"]}`)
	broadcast := func() {
		hub.Broadcast(service.ChannelRunningTrade, "BBCA", envelope("BBCA"))
		hub.Broadcast(service.ChannelRunningTrade, "ANTM", envelope("ANTM"))
	}
	before := fr.pumpBroadcasts(broadcast, func(got [][]byte) bool {
		syms := map[string]bool{}
		for _, raw := range got {
			syms[decode(t, raw).Symbol] = true
		}
		return syms["BBCA"] && syms["ANTM"]
	}, 3*time.Second)
	require.NotEmpty(t, before)

	send(t, conn, `{"action":"unsubscribe","channel":"running_trade","symbols":["BBCA"]}`)
	time.Sleep(100 * time.Millisecond) // let the read pump apply it

	seen := map[string]int{}
	for range 5 {
		broadcast()
		time.Sleep(20 * time.Millisecond)
		for _, raw := range fr.window(50 * time.Millisecond) {
			seen[decode(t, raw).Symbol]++
		}
	}
	time.Sleep(200 * time.Millisecond)
	for _, raw := range fr.window(300 * time.Millisecond) {
		seen[decode(t, raw).Symbol]++
	}
	assert.Zero(t, seen["BBCA"], "unsubscribed symbol no longer delivered")
	assert.NotZero(t, seen["ANTM"], "remaining symbol still delivered")
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

// Spec protocol: {"action":"subscribe","symbols":["BBCA","BBRI"]} — no
// "channel" key exists in the spec, and the data envelope is
// {"type":"running_trade","symbol":"BBCA","data":[…]}.
func TestSpecConformantSubscribeWithoutChannel(t *testing.T) {
	_, hub, conn, fr := newTestServer(t)

	send(t, conn, `{"action":"subscribe","symbols":["BBCA"]}`)
	// Wait until the read pump has applied the control frame: keep
	// broadcasting; while the client is still in default-broadcast every
	// symbol passes, once applied only BBCA does. Assert only on the tail
	// after the last ANTM (pre-apply frame).
	frames := fr.pumpBroadcasts(
		func() {
			hub.Broadcast(service.ChannelRunningTrade, "BBCA", envelope("BBCA"))
			hub.Broadcast(service.ChannelRunningTrade, "ANTM", envelope("ANTM"))
		},
		func(f [][]byte) bool { return len(f) >= 4 },
		3*time.Second,
	)
	require.NotEmpty(t, frames, "spec-conformant subscribe must be accepted")
	lastANTM := -1
	for i, raw := range frames {
		if decode(t, raw).Symbol == "ANTM" {
			lastANTM = i
		}
	}
	require.Less(t, lastANTM, len(frames)-1, "expected at least one post-subscribe BBCA frame")
	for _, raw := range frames[lastANTM+1:] {
		assert.Equal(t, "BBCA", decode(t, raw).Symbol, "only the subscribed symbol may pass")
	}
}

func TestDataEnvelopeTypeIsRunningTrade(t *testing.T) {
	// The envelope type is owned by the service layer's serializer; the
	// canonical assertion lives in service/ingest_test.go (env.Type=="running_trade"
	// through the real decode path). Here we assert the delivery-layer
	// contract: frames pass through untouched, so clients see exactly the
	// spec shape {"type":"running_trade","symbol":...,"data":[...]}.
	_, hub, conn, fr := newTestServer(t)

	send(t, conn, `{"action":"subscribe","symbols":["BBCA"]}`)
	time.Sleep(100 * time.Millisecond)

	frames := fr.pumpBroadcasts(
		func() { hub.Broadcast(service.ChannelRunningTrade, "BBCA", envelope("BBCA")) },
		func(f [][]byte) bool { return len(f) > 0 },
		3*time.Second,
	)
	require.NotEmpty(t, frames)
	env := decode(t, frames[0])
	assert.Equal(t, "running_trade", env.Type)
	assert.Equal(t, "BBCA", env.Symbol)
}

// Spec testing note: "ping/pong keepalive". The server pings every 45s and
// renews the read deadline on pong. Waiting 45s in a test is unreasonable, so
// assert the timing contract on the exported constants instead: the ping
// period must sit inside the read deadline so a live connection never times
// out, with margin for one missed ping.
func TestKeepaliveTimingContract(t *testing.T) {
	assert.Equal(t, 45*time.Second, service.PingPeriod(), "spec fixes the ping ticker at ±45s")
	assert.Less(t, service.PingPeriod(), service.PongWait,
		"ping must fire strictly inside the read deadline")
	assert.GreaterOrEqual(t, service.PongWait-service.PingPeriod(), 10*time.Second,
		"enough margin to survive one lost ping")
}
