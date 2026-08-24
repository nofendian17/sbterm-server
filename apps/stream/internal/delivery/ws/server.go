// Package ws serves the public WebSocket fan-out endpoint: it upgrades
// connections, applies client subscriptions to the shared hub, and keeps
// connections healthy.
package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"

	"github.com/nofendian17/sbterm/apps/stream/internal/service"
)

// readLimit bounds one inbound control frame; subscribe lists are small.
const readLimit = 4096

// Server upgrades connections and bridges them onto the hub.
type Server struct {
	hub      *service.Hub
	upgrader websocket.Upgrader
}

// NewServer builds the WebSocket delivery layer over the shared hub.
func NewServer(hub *service.Hub) *Server {
	return &Server{
		hub: hub,
		upgrader: websocket.Upgrader{
			// The endpoint streams public market data and holds no
			// credentials, so any browser origin may connect.
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

// Router assembles the HTTP surface: GET /ws upgrades to the fan-out stream,
// GET /healthz answers for orchestrator health checks.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Get("/ws", s.serveWS)
	r.Get("/healthz", s.health)
	return r
}

// serveWS upgrades, registers the client, and runs the read pump for the life
// of the connection. The write pump runs on its own goroutine; unregistering
// releases it through the hub.
func (s *Server) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrade already wrote the HTTP error
	}
	client := service.NewClient(s.hub, conn)
	s.hub.Register(client)
	go client.WritePump()

	defer func() {
		s.hub.Unregister(client)
		_ = conn.Close()
	}()

	conn.SetReadLimit(readLimit)
	_ = conn.SetReadDeadline(time.Now().Add(service.PongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(service.PongWait))
	})

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return // includes close frame, deadline expiry, and oversized frames
		}
		s.handleControl(client, raw)
	}
}

// handleControl parses one client frame and applies it. Rejected input gets
// an error envelope; the connection stays open either way.
func (s *Server) handleControl(client *service.Client, raw []byte) {
	var msg inboundMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		client.Deliver(marshalError("invalid message"))
		return
	}

	switch msg.Action {
	case actionSubscribe, actionUnsubscribe:
	default:
		client.Deliver(marshalError(fmt.Sprintf("unknown action %q", msg.Action)))
		return
	}

	channel := service.Channel(msg.Channel)
	if !service.KnownChannel(channel) {
		client.Deliver(marshalError(fmt.Sprintf("unknown channel %q", msg.Channel)))
		return
	}

	if msg.Action == actionSubscribe {
		client.Subscribe(channel, msg.Symbols)
		return
	}
	client.Unsubscribe(channel, msg.Symbols)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
