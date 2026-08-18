package httpclient

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientOptions(t *testing.T) {
	tests := []struct {
		name string
		call func(t *testing.T)
	}{
		{
			name: "with http client uses injected doer",
			call: func(t *testing.T) {
				var called atomic.Bool
				c := NewClient(WithHTTPClient(doerFunc(func(req *http.Request) (*http.Response, error) {
					called.Store(true)
					assert.Equal(t, http.MethodGet, req.Method)
					return &http.Response{
						StatusCode: http.StatusAccepted,
						Body:       io.NopCloser(strings.NewReader("custom")),
						Header:     make(http.Header),
						Request:    req,
					}, nil
				})))

				res, err := c.Do(httptest.NewRequest(http.MethodGet, "http://example.com", nil))
				require.NoError(t, err)
				defer res.Body.Close()
				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)

				assert.True(t, called.Load())
				assert.Equal(t, http.StatusAccepted, res.StatusCode)
				assert.Equal(t, "custom", string(body))
			},
		},
		{
			name: "with timeout aborts slow responses",
			call: func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(50 * time.Millisecond)
					fmt.Fprint(w, "late")
				}))
				defer srv.Close()

				c := NewClient(WithTimeout(time.Millisecond))
				_, err := c.Get(srv.URL, nil)
				assert.Error(t, err)
			},
		},
		{
			name: "with exponential backoff retries",
			call: func(t *testing.T) {
				c := NewClient(
					WithRetryCount(1),
					WithExponentialBackoff(time.Millisecond, 2*time.Millisecond, 2, time.Millisecond),
				)
				req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:1", nil)
				_, err := c.Do(req)
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.call(t)
		})
	}
}

func TestClient(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
		req  func(t *testing.T, c Client, url string)
	}{
		{
			name: "get returns body and 200",
			req: func(t *testing.T, c Client, url string) {
				res, err := c.Get(url, nil)
				require.NoError(t, err)
				defer res.Body.Close()
				assert.Equal(t, http.StatusOK, res.StatusCode)
				body, _ := io.ReadAll(res.Body)
				assert.Equal(t, "hello", string(body))
			},
		},
		{
			name: "get forwards headers",
			req: func(t *testing.T, c Client, url string) {
				headers := http.Header{}
				headers.Set("X-Api-Key", "secret")
				res, err := c.Get(url, headers)
				require.NoError(t, err)
				defer res.Body.Close()
				body, _ := io.ReadAll(res.Body)
				assert.Equal(t, "secret", string(body))
			},
		},
		{
			name: "post sends body and headers",
			req: func(t *testing.T, c Client, url string) {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				res, err := c.Post(url, headers, strings.NewReader("payload"))
				require.NoError(t, err)
				defer res.Body.Close()
				assert.Equal(t, http.StatusOK, res.StatusCode)
				body, _ := io.ReadAll(res.Body)
				assert.Equal(t, "payload", string(body))
			},
		},
		{
			name: "retries transport errors until success",
			opts: []Option{
				WithRetryCount(3),
				WithConstantBackoff(time.Millisecond, 2*time.Millisecond),
			},
			req: func(t *testing.T, c Client, url string) {
				res, err := c.Get(url, nil)
				require.NoError(t, err)
				defer res.Body.Close()
				body, _ := io.ReadAll(res.Body)
				assert.Equal(t, "recovered", string(body))
			},
		},
		{
			name: "non-2xx is returned without retry",
			opts: []Option{
				WithRetryCount(3),
				WithConstantBackoff(time.Millisecond, 2*time.Millisecond),
			},
			req: func(t *testing.T, c Client, url string) {
				res, err := c.Get(url, nil)
				require.NoError(t, err)
				defer res.Body.Close()
				assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var count atomic.Int32
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := count.Add(1)
				switch tt.name {
				case "retries transport errors until success":
					if n < 3 {
						hj, ok := w.(http.Hijacker)
						if !ok {
							t.Errorf("server does not support hijacking")
							return
						}
						conn, _, _ := hj.Hijack()
						conn.Close()
						return
					}
					fmt.Fprint(w, "recovered")
				case "non-2xx is returned without retry":
					http.Error(w, "boom", http.StatusInternalServerError)
				case "get forwards headers":
					fmt.Fprint(w, r.Header.Get("X-Api-Key"))
				case "post sends body and headers":
					body, _ := io.ReadAll(r.Body)
					w.Write(body)
				default:
					fmt.Fprint(w, "hello")
				}
			})

			srv := httptest.NewServer(handler)
			defer srv.Close()

			c := NewClient(tt.opts...)
			tt.req(t, c, srv.URL)
		})
	}
}
