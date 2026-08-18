package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const corpActionBody = `{"message":"Successfully get corp action","data":[{"action_type":"rups","action_info":{"rups":{"company_id":"105","company_symbol":"BUVA","corp_action_active":false,"rups_created":"2026-05-20","rups_date":"2026-06-11","rups_id":"1460868","rups_time":"14:00"}}},{"action_type":"rightissue","action_info":{"rightissue":{"company_id":"105","company_symbol":"BUVA-R","rightissue_id":"11504","rightissue_price":150,"rightissue_price_formatted":"","rightissue_ratio":"225 : 44","rightissue_exdate":"2025-11-04"}}}]}`

func TestGetCorpActions(t *testing.T) {
	tests := []struct {
		name    string
		symbol  string
		limit   int
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *CorpActionResponse)
	}{
		{
			name:   "returns corp actions and preserves action info",
			symbol: "BUVA",
			limit:  30,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/corpaction/BUVA", r.URL.Path)
				assert.Equal(t, "30", r.URL.Query().Get("limit"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(corpActionBody))
			},
			check: func(t *testing.T, resp *CorpActionResponse) {
				require.Len(t, resp.Data, 2)
				assert.Equal(t, "rups", resp.Data[0].ActionType)
				require.NotNil(t, resp.Data[0].Rups)
				assert.Equal(t, "2026-06-11", resp.Data[0].Rups.RupsDate)
				assert.Equal(t, "14:00", resp.Data[0].Rups.RupsTime)
				assert.Equal(t, "rightissue", resp.Data[1].ActionType)
				require.NotNil(t, resp.Data[1].RightIssue)
				assert.Equal(t, int64(150), int64(resp.Data[1].RightIssue.RightIssuePrice))
				assert.Equal(t, "225 : 44", resp.Data[1].RightIssue.RightIssueRatio)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetCorpActions(context.Background(), tt.symbol, tt.limit)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
