package watchlist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/middleware"
	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

func TestWatchlistHandler_List(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		setup    func(uc *mocks.MockWatchlistUsecase)
		wantCode int
	}{
		{
			name:   "success",
			userID: "u1",
			setup: func(uc *mocks.MockWatchlistUsecase) {
				uc.EXPECT().List(gomock.Any(), "u1").Return([]domain.Watchlist{
					{Symbol: "BBCA"},
				}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name:     "unauthorized",
			userID:   "",
			setup:    func(uc *mocks.MockWatchlistUsecase) {},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:   "internal error",
			userID: "u1",
			setup: func(uc *mocks.MockWatchlistUsecase) {
				uc.EXPECT().List(gomock.Any(), "u1").Return(nil, errors.New("db error"))
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc := mocks.NewMockWatchlistUsecase(ctrl)
			tt.setup(uc)

			handler := NewWatchlistHandler(uc, validator.New())
			req := httptest.NewRequest(http.MethodGet, "/api/v1/watchlists", nil)
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), middleware.CtxUserID, tt.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()
			handler.List(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestWatchlistHandler_Add(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		body     any
		setup    func(uc *mocks.MockWatchlistUsecase)
		wantCode int
	}{
		{
			name:   "success",
			userID: "u1",
			body:   addWatchlistRequest{Symbol: "BBCA", Label: "Bank"},
			setup: func(uc *mocks.MockWatchlistUsecase) {
				uc.EXPECT().Add(gomock.Any(), "u1", domain.AddWatchlistInput{Symbol: "BBCA", Label: "Bank"}).Return(nil)
			},
			wantCode: http.StatusCreated,
		},
		{
			name:     "unauthorized",
			userID:   "",
			body:     addWatchlistRequest{Symbol: "BBCA"},
			setup:    func(uc *mocks.MockWatchlistUsecase) {},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "invalid body",
			userID:   "u1",
			body:     "not json",
			setup:    func(uc *mocks.MockWatchlistUsecase) {},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "validation failed - empty symbol",
			userID:   "u1",
			body:     addWatchlistRequest{Symbol: "", Label: "Bank"},
			setup:    func(uc *mocks.MockWatchlistUsecase) {},
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:   "duplicate",
			userID: "u1",
			body:   addWatchlistRequest{Symbol: "BBCA"},
			setup: func(uc *mocks.MockWatchlistUsecase) {
				uc.EXPECT().Add(gomock.Any(), "u1", domain.AddWatchlistInput{Symbol: "BBCA"}).Return(domain.ErrDuplicateWatchlist)
			},
			wantCode: http.StatusConflict,
		},
		{
			name:   "stock not in catalog",
			userID: "u1",
			body:   addWatchlistRequest{Symbol: "ZZZZ"},
			setup: func(uc *mocks.MockWatchlistUsecase) {
				uc.EXPECT().Add(gomock.Any(), "u1", domain.AddWatchlistInput{Symbol: "ZZZZ"}).Return(domain.ErrStockNotFound)
			},
			wantCode: http.StatusNotFound,
		},
		{
			name:   "internal error",
			userID: "u1",
			body:   addWatchlistRequest{Symbol: "BBCA"},
			setup: func(uc *mocks.MockWatchlistUsecase) {
				uc.EXPECT().Add(gomock.Any(), "u1", domain.AddWatchlistInput{Symbol: "BBCA"}).Return(errors.New("db error"))
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc := mocks.NewMockWatchlistUsecase(ctrl)
			tt.setup(uc)

			handler := NewWatchlistHandler(uc, validator.New())
			var body bytes.Buffer
			if s, ok := tt.body.(string); ok {
				body.WriteString(s)
			} else {
				json.NewEncoder(&body).Encode(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlists", &body)
			req.Header.Set("Content-Type", "application/json")
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), middleware.CtxUserID, tt.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()
			handler.Add(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestWatchlistHandler_Remove(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		symbol   string
		setup    func(uc *mocks.MockWatchlistUsecase)
		wantCode int
	}{
		{
			name:   "success",
			userID: "u1",
			symbol: "BBCA",
			setup: func(uc *mocks.MockWatchlistUsecase) {
				uc.EXPECT().Remove(gomock.Any(), "u1", "BBCA").Return(nil)
			},
			wantCode: http.StatusNoContent,
		},
		{
			name:     "unauthorized",
			userID:   "",
			symbol:   "BBCA",
			setup:    func(uc *mocks.MockWatchlistUsecase) {},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:   "internal error",
			userID: "u1",
			symbol: "BBCA",
			setup: func(uc *mocks.MockWatchlistUsecase) {
				uc.EXPECT().Remove(gomock.Any(), "u1", "BBCA").Return(errors.New("db error"))
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc := mocks.NewMockWatchlistUsecase(ctrl)
			tt.setup(uc)

			handler := NewWatchlistHandler(uc, validator.New())
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/watchlists/"+tt.symbol, nil)
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), middleware.CtxUserID, tt.userID)
				req = req.WithContext(ctx)
			}
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("symbol", tt.symbol)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rec := httptest.NewRecorder()
			handler.Remove(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestWatchlistHandler_ListByAdmin(t *testing.T) {
	tests := []struct {
		name       string
		pathID     string
		callerID   string
		setup      func(uc *mocks.MockWatchlistUsecase)
		wantCode   int
		wantCalled bool
	}{
		{
			name:     "success - admin views target user watchlists",
			pathID:   "target-uuid",
			callerID: "admin-uuid",
			setup: func(uc *mocks.MockWatchlistUsecase) {
				uc.EXPECT().List(gomock.Any(), "target-uuid").Return([]domain.Watchlist{
					{Symbol: "BBCA"},
				}, nil)
			},
			wantCode:   http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "missing path id",
			pathID:     "",
			callerID:   "admin-uuid",
			setup:      func(uc *mocks.MockWatchlistUsecase) {},
			wantCode:   http.StatusBadRequest,
			wantCalled: false,
		},
		{
			name:     "internal error",
			pathID:   "target-uuid",
			callerID: "admin-uuid",
			setup: func(uc *mocks.MockWatchlistUsecase) {
				uc.EXPECT().List(gomock.Any(), "target-uuid").Return(nil, errors.New("db error"))
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "path id overrides caller id (regression for IDOR-like misdirection)",
			pathID:   "other-user",
			callerID: "admin-caller",
			setup: func(uc *mocks.MockWatchlistUsecase) {
				uc.EXPECT().List(gomock.Any(), "other-user").Return([]domain.Watchlist{
					{Symbol: "TLKM"},
				}, nil)
			},
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc := mocks.NewMockWatchlistUsecase(ctrl)
			tt.setup(uc)

			handler := NewWatchlistHandler(uc, validator.New())
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/"+tt.pathID+"/watchlists", nil)
			if tt.callerID != "" {
				ctx := context.WithValue(req.Context(), middleware.CtxUserID, tt.callerID)
				req = req.WithContext(ctx)
			}
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.pathID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rec := httptest.NewRecorder()
			handler.ListByAdmin(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}
