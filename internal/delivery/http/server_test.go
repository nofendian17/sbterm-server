package http

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServerOptions(t *testing.T) {
	tests := []struct {
		name      string
		opts      []Option
		wantAddr  string
		wantRead  time.Duration
		wantWrite time.Duration
		wantIdle  time.Duration
	}{
		{
			name:      "all options are applied",
			opts:      []Option{WithAddr("127.0.0.1:0"), WithReadTimeout(time.Second), WithWriteTimeout(2 * time.Second), WithIdleTimeout(3 * time.Second)},
			wantAddr:  "127.0.0.1:0",
			wantRead:  time.Second,
			wantWrite: 2 * time.Second,
			wantIdle:  3 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			srv := NewServer(handler, tt.opts...)

			assert.NotNil(t, srv.Handler())
			assert.Equal(t, tt.wantAddr, srv.httpServer.Addr)
			assert.Equal(t, tt.wantRead, srv.httpServer.ReadTimeout)
			assert.Equal(t, tt.wantWrite, srv.httpServer.WriteTimeout)
			assert.Equal(t, tt.wantIdle, srv.httpServer.IdleTimeout)
		})
	}
}

func TestServerShutdown(t *testing.T) {
	tests := []struct {
		name    string
		handler http.Handler
		during  func(t *testing.T, addr string)
		check   func(t *testing.T, elapsed time.Duration)
	}{
		{
			name: "drains in flight request before returning",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(300 * time.Millisecond)
				fmt.Fprint(w, "done")
			}),
			during: func(t *testing.T, addr string) {
				resp, err := http.Get("http://" + addr)
				require.NoError(t, err)
				defer resp.Body.Close()
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				body, _ := io.ReadAll(resp.Body)
				assert.Equal(t, "done", string(body))
			},
			check: func(t *testing.T, elapsed time.Duration) {
				assert.GreaterOrEqual(t, elapsed, 150*time.Millisecond,
					"Shutdown() should wait for the in-flight request to complete")
			},
		},
		{
			name:    "returns promptly when idle",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") }),
			check: func(t *testing.T, elapsed time.Duration) {
				assert.Less(t, elapsed, 100*time.Millisecond,
					"Shutdown() should return quickly with no in-flight requests")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(tt.handler)

			ln, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			served := make(chan struct{})
			go func() {
				defer close(served)
				_ = srv.httpServer.Serve(ln)
			}()

			var duringDone <-chan struct{}
			if tt.during != nil {
				done := make(chan struct{})
				go func() {
					defer close(done)
					tt.during(t, ln.Addr().String())
				}()
				duringDone = done
				time.Sleep(100 * time.Millisecond)
			}

			start := time.Now()
			err = srv.Shutdown()
			elapsed := time.Since(start)
			require.NoError(t, err)

			if duringDone != nil {
				select {
				case <-duringDone:
				case <-time.After(2 * time.Second):
					t.Fatal("in-flight request did not complete after shutdown")
				}
			}
			<-served

			tt.check(t, elapsed)
		})
	}
}
