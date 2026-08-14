package repository

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
)

const userStreamRepoBody = `{"data":{"stream":[{"stream_id":123,"title_url":"https://stockbit.com/post/123","title":"Thesis","content":"content","created_at":"2026-08-15T00:00:00Z","created_display":"2 hours ago","updated_at":"2026-08-15T00:00:00Z","user":{"user_id":1,"is_author":true,"username":"StockbitReports","fullname":"Full Name","avatar":"https://stockbit.com/avatar.png","is_verified":true,"user_privilege":"STREAM_USER_PRIVILEGE_PRO","is_pro":true,"country":"ID","verified_status":"STREAM_USER_VERIFIED_STATUS_VERIFIED"},"status":{"is_pinned":true,"is_trending":true,"is_liked":true},"total_replies":5,"total_likes":10,"type":"STREAM_TYPE_POST","parent_stream_id":999,"reports":[{"type":"STREAM_REPORT_TYPE_SPAM"}],"topics":["trading"],"summary":{"title":"Summary","summary":"summary text","key_points":["kp1","kp2"],"key_takeaway":"takeaway","model":"gpt","model_version":"1"},"reaction":{"reactions":[{"reaction":"LIKE","total":3}],"total":3,"my_reaction":"LIKE"}}],"pagination":{"is_last_page":true,"next_cursor":456,"total":1},"invalid_watchlist_ids":["bad-symbol"]}}`

const userStreamAnnouncementRepoBody = `{"message":"Successfully retrieved stream announcement","data":[{"id":3547541,"company_id":497,"posted_on":"2026-08-15 07:41:51","headline":"Rencana Transaksi Material Dengan Persetujuan RUPS (KOREKSI) [SILO]","title":"f-32120989-0_SILO.pdf","attachment":"https://emitten-announcement.stockbit.com/attachments/f-32120989-0_SILO.pdf","retrieved_on":"2026-08-15 00:50:17","symbol":"SILO","name":"Siloam International Hospitals Tbk","company_icon_url":"https://assets.stockbit.com/logos/companies/SILO.png"}]}`

func TestStreamRepositoryGetStreamAnnouncement(t *testing.T) {
	const streamID = "f3e83a0aeb3c9c48800b7f3beafc8aba"
	tests := []struct {
		name     string
		status   int
		body     string
		wantErr  bool
		wantUp   bool
		wantCode int
	}{
		{
			name:   "returns mapped stream announcements",
			status: http.StatusOK,
			body:   userStreamAnnouncementRepoBody,
		},
		{
			name:    "propagates upstream error",
			status:  http.StatusInternalServerError,
			body:    `{"message":"boom"}`,
			wantErr: true,
		},
		{
			name:     "translates upstream 400 into domain error",
			status:   http.StatusBadRequest,
			body:     `{"message":"Please check your request"}`,
			wantErr:  true,
			wantUp:   true,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/stream/announcement/"+streamID, r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewStreamRepository(client)

			got, err := repo.GetStreamAnnouncement(context.Background(), streamID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantUp {
					var up *domain.UpstreamError
					require.ErrorAs(t, err, &up)
					assert.Equal(t, tt.wantCode, up.Status)
				}
				return
			}
			require.NoError(t, err)
			require.Len(t, got, 1)
			a := got[0]
			assert.Equal(t, int64(3547541), a.ID)
			assert.Equal(t, int64(497), a.CompanyID)
			assert.Equal(t, "Rencana Transaksi Material Dengan Persetujuan RUPS (KOREKSI) [SILO]", a.Headline)
			assert.Equal(t, "f-32120989-0_SILO.pdf", a.Title)
			assert.Equal(t, "https://emitten-announcement.stockbit.com/attachments/f-32120989-0_SILO.pdf", a.Attachment)
			assert.Equal(t, "SILO", a.Symbol)
			assert.Equal(t, "Siloam International Hospitals Tbk", a.Name)
			assert.Equal(t, "https://assets.stockbit.com/logos/companies/SILO.png", a.CompanyIconURL)
		})
	}
}

func TestStreamRepositoryGetUserStream(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantErr  bool
		wantUp   bool
		wantCode int
	}{
		{
			name:   "returns mapped user stream",
			status: http.StatusOK,
			body:   userStreamRepoBody,
		},
		{
			name:    "propagates upstream error",
			status:  http.StatusInternalServerError,
			body:    `{"message":"boom"}`,
			wantErr: true,
		},
		{
			name:     "translates upstream 400 into domain error",
			status:   http.StatusBadRequest,
			body:     `{"message":"Please check your request"}`,
			wantErr:  true,
			wantUp:   true,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/stream/v3/user/StockbitReports", r.URL.Path)

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				var req struct {
					Category     string `json:"category"`
					LastStreamID int64  `json:"last_stream_id"`
					Limit        int    `json:"limit"`
				}
				require.NoError(t, json.Unmarshal(body, &req))
				assert.Equal(t, "STREAM_CATEGORY_MAIN_IDEAS", req.Category)
				assert.Equal(t, int64(34884782), req.LastStreamID)
				assert.Equal(t, 20, req.Limit)

				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewStreamRepository(client)

			got, err := repo.GetUserStream(context.Background(), "StockbitReports", "STREAM_CATEGORY_MAIN_IDEAS", 34884782, 20)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantUp {
					var up *domain.UpstreamError
					require.ErrorAs(t, err, &up)
					assert.Equal(t, tt.wantCode, up.Status)
				}
				return
			}
			require.NoError(t, err)
			require.Len(t, got.Stream, 1)
			p := got.Stream[0]
			assert.Equal(t, int64(123), p.StreamID)
			assert.Equal(t, "Thesis", p.Title)
			assert.True(t, p.User.IsAuthor)
			assert.Equal(t, "StockbitReports", p.User.Username)
			assert.True(t, p.Status.IsPinned)
			assert.False(t, p.Status.IsDeleted)
			assert.Equal(t, 10, p.TotalLikes)
			assert.Equal(t, int64(999), p.ParentStreamID)
			require.Len(t, p.Reports, 1)
			assert.Equal(t, "STREAM_REPORT_TYPE_SPAM", p.Reports[0].Type)
			assert.Equal(t, []string{"trading"}, p.Topics)
			require.NotNil(t, p.Summary)
			assert.Equal(t, []string{"kp1", "kp2"}, p.Summary.KeyPoints)
			require.NotNil(t, p.Reaction)
			require.Len(t, p.Reaction.Reactions, 1)
			assert.Equal(t, "LIKE", p.Reaction.Reactions[0].Reaction)
			assert.Equal(t, 3, p.Reaction.Total)
			assert.True(t, got.Pagination.IsLastPage)
			assert.Equal(t, int64(456), got.Pagination.NextCursor)
			assert.Equal(t, 1, got.Pagination.Total)
			assert.Equal(t, []string{"bad-symbol"}, got.InvalidWatchlistIDs)
		})
	}
}
