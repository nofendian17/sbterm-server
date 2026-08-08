package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAuth struct {
	token     string
	refreshed atomic.Int32
}

func (s *stubAuth) EnsureToken(ctx context.Context) (string, error) { return s.token, nil }
func (s *stubAuth) Refresh(ctx context.Context) (string, error) {
	s.refreshed.Add(1)
	s.token = "at-new"
	return s.token, nil
}

func TestDoAttachesAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	err := New(
		WithBaseURL(srv.URL),
		WithAuthenticator(&stubAuth{token: "at-ok"}),
	).Get(context.Background(), "/v1/stocks", nil, nil)
	require.NoError(t, err)
}

func TestLoginSendsNoAuthorization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, loginPath, r.URL.Path)
		assert.Empty(t, r.Header.Get("Authorization"))
		writeLoginResponse(w, "at-1", "rt-1")
	}))
	defer srv.Close()

	_, err := New(
		WithBaseURL(srv.URL),
		WithAuthenticator(&stubAuth{token: "at-ok"}),
	).Login(context.Background(), LoginRequest{User: "budi", Password: "secret"})
	require.NoError(t, err)
}

func TestRefreshSendsRefreshTokenNotAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, refreshPath, r.URL.Path)
		assert.Equal(t, "Bearer rt-1", r.Header.Get("Authorization"))
		writeRefreshResponse(w, "at-2", "rt-2")
	}))
	defer srv.Close()

	_, err := New(
		WithBaseURL(srv.URL),
		WithAuthenticator(&stubAuth{token: "at-ok"}),
	).Refresh(context.Background(), "rt-1")
	require.NoError(t, err)
}

func TestDoRefreshesAndRetriesOnUnauthorized(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/stocks", r.URL.Path)
		if hits.Add(1) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		assert.Equal(t, "Bearer at-new", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	auth := &stubAuth{token: "at-old"}
	err := New(
		WithBaseURL(srv.URL),
		WithAuthenticator(auth),
	).Get(context.Background(), "/v1/stocks", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int32(1), auth.refreshed.Load())
	assert.Equal(t, int32(2), hits.Load())
}

func TestDoDoesNotRetryWhenRefreshFails(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	auth := &stubAuth{token: "at-old"}
	err := New(
		WithBaseURL(srv.URL),
		WithAuthenticator(auth),
	).Get(context.Background(), "/v1/stocks", nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
	assert.Equal(t, int32(2), hits.Load())
}
