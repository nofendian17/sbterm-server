package stockapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ListSymbols_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/stocks", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": []map[string]any{
				{"symbol": "BBCA", "name": "Bank Central Asia Tbk.", "icon_url": "https://x/BBCA.png", "company_status": "STATUS_ACTIVE"},
				{"symbol": "TLKM", "name": "Telkom Indonesia Tbk.", "icon_url": "", "company_status": "STATUS_INACTIVE"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	got, err := c.ListSymbols(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "BBCA", got[0].Symbol)
	assert.True(t, got[0].IsActive)
	require.NotNil(t, got[0].IconURL)
	assert.Equal(t, "https://x/BBCA.png", *got[0].IconURL)
	assert.False(t, got[1].IsActive) // company_status != STATUS_ACTIVE
	assert.Nil(t, got[1].IconURL)
}

func TestClient_ListSymbols_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.ListSymbols(context.Background())
	assert.Error(t, err)
}

func TestClient_ListSymbols_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.ListSymbols(context.Background())
	assert.Error(t, err)
}
