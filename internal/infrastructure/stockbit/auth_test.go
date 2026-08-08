package stockbit

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/login/v6/username", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, _ := io.ReadAll(r.Body)
		assert.JSONEq(t, `{"player_id":"p123","user":"budi","password":"secret"}`, string(body))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"You have been successfully logged in","data":{"login":{"user":{"id":7377894,"username":"bennyanwar","email":"b@example.com","fullname":"","avatar":"https://avatar.stockbit.com/x.png","country":"ID","sns":{"facebook":false,"apple":false,"google":true},"has_password_been_set":true,"is_phone_verified":true,"is_verified":true,"privilege":{"name":"PRIVILEGE_MEMBER","code":0},"watchlist_id":16091024,"exchange":"ID"},"token_data":{"access":{"token":"at1","expired_at":"2026-01-01T00:00:00Z"},"refresh":{"token":"rt1","expired_at":"2026-02-01T00:00:00Z"}}},"support":{"id":"0b4104d1-aadc-4a0e-8df9-0f535180475b"}}}`))
	}))
	defer srv.Close()

	resp, err := New(WithBaseURL(srv.URL)).Login(context.Background(), LoginRequest{
		PlayerID: "p123",
		User:     "budi",
		Password: "secret",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "You have been successfully logged in", resp.Message)
	assert.Equal(t, int64(7377894), resp.Data.Login.User.ID)
	assert.Equal(t, "bennyanwar", resp.Data.Login.User.Username)
	assert.Equal(t, "ID", resp.Data.Login.User.Country)
	assert.True(t, resp.Data.Login.User.SNS.Google)
	assert.True(t, resp.Data.Login.User.IsVerified)
	assert.Equal(t, "PRIVILEGE_MEMBER", resp.Data.Login.User.Privilege.Name)
	assert.Equal(t, int64(16091024), resp.Data.Login.User.WatchlistID)
	assert.Equal(t, "0b4104d1-aadc-4a0e-8df9-0f535180475b", resp.Data.Support.ID)
	assert.Equal(t, "at1", resp.Data.Login.TokenData.Access.Token)
	assert.Equal(t, "rt1", resp.Data.Login.TokenData.Refresh.Token)
}

func TestLogout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/logout", r.URL.Path)
		assert.Equal(t, "Bearer at1", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"You have been successfully logged out"}`))
	}))
	defer srv.Close()

	resp, err := New(WithBaseURL(srv.URL)).Logout(context.Background(), "at1")
	require.NoError(t, err)
	assert.Equal(t, "You have been successfully logged out", resp.Message)
}

func TestRefresh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/login/refresh", r.URL.Path)
		assert.Equal(t, "Bearer rt1", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"You have been successfully refresh token","data":{"access":{"token":"tok2","expired_at":"2026-01-01T00:00:00Z"},"refresh":{"token":"rt2","expired_at":"2026-02-01T00:00:00Z"}}}`))
	}))
	defer srv.Close()

	resp, err := New(WithBaseURL(srv.URL)).Refresh(context.Background(), "rt1")
	require.NoError(t, err)
	assert.Equal(t, "tok2", resp.Data.Access.Token)
	assert.Equal(t, "rt2", resp.Data.Refresh.Token)
	assert.Equal(t, "You have been successfully refresh token", resp.Message)
}