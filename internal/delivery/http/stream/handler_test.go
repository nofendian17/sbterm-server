package stream

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/mocks"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

func TestStreamHandlerStreamAnnouncement(t *testing.T) {
	const streamID = "f3e83a0aeb3c9c48800b7f3beafc8aba"
	tests := []struct {
		name         string
		path         string
		direct       bool
		setup        func(uc *mocks.MockStreamUsecase)
		wantStatus   int
		wantErrCode  string
		wantHeadline string
	}{
		{
			name: "returns stream announcements",
			path: "/v1/stream/announcement/" + streamID,
			setup: func(uc *mocks.MockStreamUsecase) {
				uc.EXPECT().GetStreamAnnouncement(gomock.Any(), streamID).Return([]domain.StreamAnnouncement{
					{ID: 3547541, CompanyID: 497, PostedOn: "2026-08-15 07:41:51", Headline: "Rencana Transaksi Material Dengan Persetujuan RUPS (KOREKSI) [SILO]", Title: "f-32120989-0_SILO.pdf", Attachment: "https://emitten-announcement.stockbit.com/attachments/f-32120989-0_SILO.pdf", RetrievedOn: "2026-08-15 00:50:17", Symbol: "SILO", Name: "Siloam International Hospitals Tbk", CompanyIconURL: "https://assets.stockbit.com/logos/companies/SILO.png"},
				}, nil)
			},
			wantStatus:   http.StatusOK,
			wantHeadline: "Rencana Transaksi Material Dengan Persetujuan RUPS (KOREKSI) [SILO]",
		},
		{
			name:        "missing stream_id returns 422",
			path:        "/v1/stream/announcement/",
			direct:      true,
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "upstream 400 returns 422",
			path: "/v1/stream/announcement/" + streamID,
			setup: func(uc *mocks.MockStreamUsecase) {
				uc.EXPECT().GetStreamAnnouncement(gomock.Any(), streamID).Return(nil, &domain.UpstreamError{Status: http.StatusBadRequest, Msg: "invalid"})
			},
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/v1/stream/announcement/" + streamID,
			setup: func(uc *mocks.MockStreamUsecase) {
				uc.EXPECT().GetStreamAnnouncement(gomock.Any(), streamID).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockStreamUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			rec := httptest.NewRecorder()
			if tt.direct {
				h := NewStreamHandler(uc, validator.New())
				h.StreamAnnouncement(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			} else {
				r := chi.NewRouter()
				h := NewStreamHandler(uc, validator.New())
				r.Get("/v1/stream/announcement/{stream_id}", h.StreamAnnouncement)
				r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			}

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    []struct {
					Headline string `json:"headline"`
				} `json:"data"`
				Error *struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))

			if tt.wantErrCode != "" {
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantErrCode, env.Error.Code)
				return
			}
			if tt.wantHeadline != "" {
				require.Len(t, env.Data, 1)
				assert.Equal(t, tt.wantHeadline, env.Data[0].Headline)
			}
		})
	}
}

func TestStreamHandlerUserStream(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		direct      bool
		setup       func(uc *mocks.MockStreamUsecase)
		wantStatus  int
		wantErrCode string
		wantTitle   string
		wantTotal   int
	}{
		{
			name: "returns stream with all params",
			path: "/v1/user/StockbitReports/stream?category=STREAM_CATEGORY_MAIN_IDEAS&last_stream_id=34884782&limit=20",
			setup: func(uc *mocks.MockStreamUsecase) {
				uc.EXPECT().GetUserStream(gomock.Any(), "StockbitReports", "STREAM_CATEGORY_MAIN_IDEAS", int64(34884782), 20).Return(&domain.UserStreamData{
					Stream: []domain.StreamPost{
						{
							StreamID:       123,
							TitleURL:       "https://stockbit.com/post/123",
							Title:          "Thesis",
							Content:        "content",
							CreatedAt:      "2026-08-15T00:00:00Z",
							CreatedDisplay: "2 hours ago",
							UpdatedAt:      "2026-08-15T00:00:00Z",
							User: domain.StreamUser{
								UserID:         1,
								IsAuthor:       true,
								Username:       "StockbitReports",
								Fullname:       "Full Name",
								Avatar:         "https://stockbit.com/avatar.png",
								IsVerified:     true,
								UserPrivilege:  "STREAM_USER_PRIVILEGE_PRO",
								IsPro:          true,
								Country:        "ID",
								VerifiedStatus: "STREAM_USER_VERIFIED_STATUS_VERIFIED",
							},
							Status:         domain.StreamStatus{IsPinned: true, IsTrending: true, IsLiked: true, IsDeleted: false},
							TotalReplies:   5,
							TotalLikes:     10,
							Type:           "STREAM_TYPE_POST",
							ParentStreamID: 0,
							Reports:        []domain.StreamReport{{Type: "STREAM_REPORT_TYPE_SPAM"}},
							Topics:         []string{"trading"},
							Summary:        &domain.StreamSummary{Title: "Summary", Summary: "summary", KeyPoints: []string{"kp"}, KeyTakeaway: "takeaway", Model: "gpt", ModelVersion: "1"},
							Reaction:       &domain.StreamReaction{Reactions: []domain.StreamReactionItem{{Reaction: "LIKE", Total: 3}}, Total: 3, MyReaction: "LIKE"},
						},
					},
					Pagination:          domain.UserStreamPaginate{IsLastPage: true, NextCursor: 456, Total: 1},
					InvalidWatchlistIDs: []string{"bad-symbol"},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantTitle:  "Thesis",
			wantTotal:  1,
		},
		{
			name: "defaults category and limit when omitted",
			path: "/v1/user/StockbitReports/stream",
			setup: func(uc *mocks.MockStreamUsecase) {
				uc.EXPECT().GetUserStream(gomock.Any(), "StockbitReports", "STREAM_CATEGORY_MAIN_IDEAS", int64(0), 20).Return(&domain.UserStreamData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "accepts news category",
			path: "/v1/user/StockbitReports/stream?category=STREAM_CATEGORY_NEWS",
			setup: func(uc *mocks.MockStreamUsecase) {
				uc.EXPECT().GetUserStream(gomock.Any(), "StockbitReports", "STREAM_CATEGORY_NEWS", int64(0), 20).Return(&domain.UserStreamData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "invalid category returns 422",
			path:        "/v1/user/StockbitReports/stream?category=BOGUS",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid last_stream_id returns 422",
			path:        "/v1/user/StockbitReports/stream?last_stream_id=abc",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "invalid limit returns 422",
			path:        "/v1/user/StockbitReports/stream?limit=abc",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "zero limit returns 422",
			path:        "/v1/user/StockbitReports/stream?limit=0",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "missing username returns 422",
			path:        "/v1/user/stream",
			direct:      true,
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "upstream 400 returns 422",
			path: "/v1/user/StockbitReports/stream",
			setup: func(uc *mocks.MockStreamUsecase) {
				uc.EXPECT().GetUserStream(gomock.Any(), "StockbitReports", "STREAM_CATEGORY_MAIN_IDEAS", int64(0), 20).Return(nil, &domain.UpstreamError{Status: http.StatusBadRequest, Msg: "invalid"})
			},
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/v1/user/StockbitReports/stream",
			setup: func(uc *mocks.MockStreamUsecase) {
				uc.EXPECT().GetUserStream(gomock.Any(), "StockbitReports", "STREAM_CATEGORY_MAIN_IDEAS", int64(0), 20).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockStreamUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			rec := httptest.NewRecorder()
			if tt.direct {
				h := NewStreamHandler(uc, validator.New())
				h.UserStream(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			} else {
				r := chi.NewRouter()
				h := NewStreamHandler(uc, validator.New())
				r.Get("/v1/user/{username}/stream", h.UserStream)
				r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			}

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Stream []struct {
						Title string `json:"title"`
					} `json:"stream"`
					Pagination struct {
						IsLastPage bool  `json:"is_last_page"`
						NextCursor int64 `json:"next_cursor"`
						Total      int   `json:"total"`
					} `json:"pagination"`
				} `json:"data"`
				Error *struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))

			if tt.wantErrCode != "" {
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantErrCode, env.Error.Code)
				return
			}
			if tt.wantTitle != "" {
				require.Len(t, env.Data.Stream, 1)
				assert.Equal(t, tt.wantTitle, env.Data.Stream[0].Title)
			}
			if tt.wantTotal != 0 {
				assert.Equal(t, tt.wantTotal, env.Data.Pagination.Total)
			}
		})
	}
}
