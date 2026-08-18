package http

import (
	"context"
	"net/http"
	"time"
)

type Option func(*options)

type options struct {
	addr         string
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
}

func WithAddr(addr string) Option {
	return func(o *options) { o.addr = addr }
}

func WithReadTimeout(d time.Duration) Option {
	return func(o *options) { o.readTimeout = d }
}

func WithWriteTimeout(d time.Duration) Option {
	return func(o *options) { o.writeTimeout = d }
}

func WithIdleTimeout(d time.Duration) Option {
	return func(o *options) { o.idleTimeout = d }
}

type Server struct {
	httpServer *http.Server
	handler    http.Handler
}

func NewServer(handler http.Handler, opts ...Option) *Server {
	o := &options{
		addr:         ":8080",
		readTimeout:  10 * time.Second,
		writeTimeout: 10 * time.Second,
		idleTimeout:  60 * time.Second,
	}
	for _, opt := range opts {
		opt(o)
	}

	return &Server{
		httpServer: &http.Server{
			Addr:         o.addr,
			Handler:      handler,
			ReadTimeout:  o.readTimeout,
			WriteTimeout: o.writeTimeout,
			IdleTimeout:  o.idleTimeout,
		},
		handler: handler,
	}
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}
