package stock

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

func testStock() domain.Stock {
	return domain.Stock{
		Symbol:    "BBCA",
		Name:      "Bank Central Asia",
		IsActive:  true,
		CreatedAt: time.Unix(1700000000, 0).UTC(),
		UpdatedAt: time.Unix(1700000000, 0).UTC(),
	}
}

func withSymbol(r *http.Request, symbol string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", symbol)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestStockHandler_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockStockUsecase(ctrl)
	uc.EXPECT().List(gomock.Any(), gomock.Any()).
		Return([]domain.Stock{testStock()}, 1, nil)

	handler := NewStockHandler(uc, validator.New())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks?page=1&limit=20", nil)
	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var env struct {
		Success bool `json:"success"`
		Data    []struct {
			Symbol string `json:"symbol"`
		} `json:"data"`
		Meta *struct {
			Page       int `json:"page"`
			Limit      int `json:"limit"`
			TotalItems int `json:"total_items"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.True(t, env.Success)
	require.Len(t, env.Data, 1)
	assert.Equal(t, "BBCA", env.Data[0].Symbol)
	require.NotNil(t, env.Meta)
	assert.Equal(t, 1, env.Meta.TotalItems)
}

func TestStockHandler_GetBySymbol(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockStockUsecase(ctrl)
		uc.EXPECT().GetBySymbol(gomock.Any(), "BBCA").Return(testStock(), nil)

		handler := NewStockHandler(uc, validator.New())
		req := withSymbol(httptest.NewRequest(http.MethodGet, "/api/v1/stocks/BBCA", nil), "BBCA")
		rec := httptest.NewRecorder()
		handler.GetBySymbol(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockStockUsecase(ctrl)
		uc.EXPECT().GetBySymbol(gomock.Any(), "ZZZZ").
			Return(domain.Stock{}, domain.ErrStockNotFound)

		handler := NewStockHandler(uc, validator.New())
		req := withSymbol(httptest.NewRequest(http.MethodGet, "/api/v1/stocks/ZZZZ", nil), "ZZZZ")
		rec := httptest.NewRecorder()
		handler.GetBySymbol(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestStockHandler_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockStockUsecase(ctrl)
		uc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(testStock(), nil)

		handler := NewStockHandler(uc, validator.New())
		body, _ := json.Marshal(createStockRequest{Symbol: "BBCA", Name: "Bank Central Asia"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/stocks", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.Create(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("duplicate", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockStockUsecase(ctrl)
		uc.EXPECT().Create(gomock.Any(), gomock.Any()).
			Return(domain.Stock{}, domain.ErrStockSymbolTaken)

		handler := NewStockHandler(uc, validator.New())
		body, _ := json.Marshal(createStockRequest{Symbol: "BBCA", Name: "Bank Central Asia"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/stocks", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.Create(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("validation failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockStockUsecase(ctrl)

		handler := NewStockHandler(uc, validator.New())
		body, _ := json.Marshal(createStockRequest{Symbol: "", Name: ""})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/stocks", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.Create(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("unknown sector", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockStockUsecase(ctrl)
		sector := "Nope"
		uc.EXPECT().Create(gomock.Any(), gomock.Any()).
			Return(domain.Stock{}, domain.ErrSectorNotFound)

		handler := NewStockHandler(uc, validator.New())
		body, _ := json.Marshal(createStockRequest{Symbol: "BBCA", Name: "Bank Central Asia", Sector: &sector})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/stocks", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.Create(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})
}

func TestStockHandler_Update(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockStockUsecase(ctrl)
		uc.EXPECT().Update(gomock.Any(), "ZZZZ", gomock.Any()).
			Return(domain.ErrStockNotFound)

		handler := NewStockHandler(uc, validator.New())
		body, _ := json.Marshal(updateStockRequest{})
		req := withSymbol(httptest.NewRequest(http.MethodPatch, "/api/v1/admin/stocks/ZZZZ", bytes.NewReader(body)), "ZZZZ")
		rec := httptest.NewRecorder()
		handler.Update(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestStockHandler_Delete(t *testing.T) {
	t.Run("has watchlists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockStockUsecase(ctrl)
		uc.EXPECT().SoftDelete(gomock.Any(), "BBCA").
			Return(domain.ErrStockHasWatchlists)

		handler := NewStockHandler(uc, validator.New())
		req := withSymbol(httptest.NewRequest(http.MethodDelete, "/api/v1/admin/stocks/BBCA", nil), "BBCA")
		rec := httptest.NewRecorder()
		handler.Delete(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestStockHandler_Sync(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockStockUsecase(ctrl)
		uc.EXPECT().SyncAll(gomock.Any()).
			Return(domain.StockSyncResult{Fetched: 2, Created: 1, Updated: 1}, nil)

		handler := NewStockHandler(uc, validator.New())
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/stocks/sync", nil)
		rec := httptest.NewRecorder()
		handler.Sync(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("upstream error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		uc := mocks.NewMockStockUsecase(ctrl)
		uc.EXPECT().SyncAll(gomock.Any()).
			Return(domain.StockSyncResult{}, errors.New("boom"))

		handler := NewStockHandler(uc, validator.New())
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/stocks/sync", nil)
		rec := httptest.NewRecorder()
		handler.Sync(rec, req)

		assert.Equal(t, http.StatusBadGateway, rec.Code)
	})
}
