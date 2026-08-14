package stockbit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm-server/pkg/log"
)

const streamBody = `{"message":"20 stream post(s) retrieved","data":{"stream":[{"stream_id":34884767,"title_url":"streams/announcement/41a33437efb2c797eb3a8e9055bb318d","title":"Laporan Harian atas Nilai Aktiva Bersih dan Komposisi Portofolio [XKMS]","content":"","created_at":"2026-08-14 18:17:44","created_display":"14 Aug 26, 18:17","updated_at":"0000-00-00 00:00:00","user":{"user_id":3,"is_author":false,"username":"StockbitReports","fullname":"Stockbit Reports","avatar":"https://avatar.stockbit.com/3-1366883879.jpeg","is_verified":true,"user_privilege":"PRIVILEGE_MEMBER","is_pro":false,"country":"COUNTRY_ID","verified_status":"VERIFIED_STATUS_COMMUNITY"},"status":{"is_pinned":false,"is_trending":false,"is_reposted":false,"is_liked":false,"is_saved":false,"is_followed":false,"is_unavailable":false,"is_junk":false,"is_spam":false,"is_violation":false,"is_deleted":false},"total_replies":0,"total_likes":0,"type":"STREAM_TYPE_REPORT","parent_stream_id":0,"reports":[{"type":"Others"}],"topics":["XKMS"],"summary":{"title":"Laporan Harian Nilai Aset Bersih","summary":"XKMS melaporkan nilai aset bersih sebesar Rp126,71 miliar per 14 Agustus 2026.","key_points":["Nilai aset bersih tercatat Rp126,71 miliar."],"key_takeaway":"Laporan harian ini bersifat rutin.","model":"SUMMARY_MODEL_AI","model_version":"v1"},"reaction":null}],"pagination":{"is_last_page":false,"next_cursor":34884487,"total":20},"invalid_watchlist_ids":[]}}`

const streamAnnouncementBody = `{"message":"Successfully retrieved stream announcement","data":[{"id":3547541,"company_id":497,"posted_on":"2026-08-15 07:41:51","headline":"Rencana Transaksi Material Dengan Persetujuan RUPS (KOREKSI) [SILO]","title":"f-32120989-0_SILO.pdf","attachment":"https://emitten-announcement.stockbit.com/attachments/f-32120989-0_SILO.pdf","retrieved_on":"2026-08-15 00:50:17","symbol":"SILO","name":"Siloam International Hospitals Tbk","company_icon_url":"https://assets.stockbit.com/logos/companies/SILO.png"}]}`

func TestGetStreamAnnouncement(t *testing.T) {
	const streamID = "f3e83a0aeb3c9c48800b7f3beafc8aba"
	tests := []struct {
		name    string
		opts    []Option
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *StreamAnnouncementResponse)
	}{
		{
			name: "returns stream announcements",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/stream/announcement/"+streamID, r.URL.Path)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(streamAnnouncementBody))
			},
			check: func(t *testing.T, resp *StreamAnnouncementResponse) {
				require.Len(t, resp.Data, 1)
				a := resp.Data[0]
				assert.Equal(t, int64(3547541), a.ID)
				assert.Equal(t, int64(497), a.CompanyID)
				assert.Equal(t, "Rencana Transaksi Material Dengan Persetujuan RUPS (KOREKSI) [SILO]", a.Headline)
				assert.Equal(t, "f-32120989-0_SILO.pdf", a.Title)
				assert.Equal(t, "https://emitten-announcement.stockbit.com/attachments/f-32120989-0_SILO.pdf", a.Attachment)
				assert.Equal(t, "SILO", a.Symbol)
				assert.Equal(t, "https://assets.stockbit.com/logos/companies/SILO.png", a.CompanyIconURL)
			},
		},
		{
			name: "uses access token",
			opts: []Option{WithAuthenticator(&stubAuth{token: "at-ok"})},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(streamAnnouncementBody))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			opts := append([]Option{WithBaseURL(srv.URL)}, tt.opts...)
			resp, err := New(opts...).GetStreamAnnouncement(context.Background(), streamID)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}

func TestGetUserStream(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		req     UserStreamRequest
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *UserStreamResponse, logs string)
	}{
		{
			name: "returns user stream posts",
			req:  UserStreamRequest{Category: "STREAM_CATEGORY_MAIN_IDEAS", LastStreamID: 34884782, Limit: 20},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/stream/v3/user/StockbitReports", r.URL.Path)
				var req UserStreamRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
				assert.Equal(t, "STREAM_CATEGORY_MAIN_IDEAS", req.Category)
				assert.Equal(t, int64(34884782), req.LastStreamID)
				assert.Equal(t, 20, req.Limit)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(streamBody))
			},
			check: func(t *testing.T, resp *UserStreamResponse, logs string) {
				require.Len(t, resp.Data.Stream, 1)
				p := resp.Data.Stream[0]
				assert.Equal(t, int64(34884767), p.StreamID)
				assert.Equal(t, "Laporan Harian atas Nilai Aktiva Bersih dan Komposisi Portofolio [XKMS]", p.Title)
				assert.Equal(t, "14 Aug 26, 18:17", p.CreatedDisplay)
				assert.Equal(t, "StockbitReports", p.User.Username)
				assert.Equal(t, "STREAM_TYPE_REPORT", p.Type)
				assert.Equal(t, []string{"XKMS"}, p.Topics)
				require.NotNil(t, p.Summary)
				assert.Equal(t, "SUMMARY_MODEL_AI", p.Summary.Model)
				assert.False(t, resp.Data.Pagination.IsLastPage)
				assert.Equal(t, int64(34884487), resp.Data.Pagination.NextCursor)
				assert.Equal(t, 20, resp.Data.Pagination.Total)
			},
		},
		{
			name: "uses access token",
			opts: []Option{WithAuthenticator(&stubAuth{token: "at-ok"})},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(streamBody))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			var buf strings.Builder
			logger := log.New(log.WithWriter(&buf), log.WithLevel(log.LevelDebug))
			opts := append([]Option{WithBaseURL(srv.URL), WithLogger(logger)}, tt.opts...)
			resp, err := New(opts...).GetUserStream(context.Background(), "StockbitReports", tt.req)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp, buf.String())
			}
		})
	}
}
