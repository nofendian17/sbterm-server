package orderbook

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

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/mocks"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

func TestOrderBookHandlerOrderBook(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setup       func(uc *mocks.MockOrderBookUsecase)
		wantStatus  int
		wantErrCode string
		wantSymbol  string
	}{
		{
			name: "returns order book",
			path: "/api/v1/company/VKTR/orderbook",
			setup: func(uc *mocks.MockOrderBookUsecase) {
				uc.EXPECT().GetOrderBook(gomock.Any(), "VKTR").Return(&domain.OrderBookData{
					Symbol:    "VKTR",
					Average:   880,
					LastPrice: 885,
					Bid:       []domain.OrderBookLevel{{Price: "880", QueNum: "138", Volume: "823200"}},
					Offer:     []domain.OrderBookLevel{{Price: "885", QueNum: "72", Volume: "497200"}},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantSymbol: "VKTR",
		},
		{
			name:        "missing symbol returns 422",
			path:        "/api/v1/company//orderbook",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "upstream 400 returns 422",
			path: "/api/v1/company/VKTR/orderbook",
			setup: func(uc *mocks.MockOrderBookUsecase) {
				uc.EXPECT().GetOrderBook(gomock.Any(), "VKTR").Return(nil, &domain.UpstreamError{Status: http.StatusBadRequest, Msg: "invalid"})
			},
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name: "usecase error returns 500",
			path: "/api/v1/company/VKTR/orderbook",
			setup: func(uc *mocks.MockOrderBookUsecase) {
				uc.EXPECT().GetOrderBook(gomock.Any(), "VKTR").Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockOrderBookUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			r := chi.NewRouter()
			h := NewOrderBookHandler(uc, validator.New())
			r.Get("/api/v1/company/{symbol}/orderbook", h.OrderBook)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    struct {
					Symbol    string `json:"symbol"`
					LastPrice int    `json:"lastprice"`
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
			if tt.wantSymbol != "" {
				assert.Equal(t, tt.wantSymbol, env.Data.Symbol)
				assert.Equal(t, 885, env.Data.LastPrice)
			}
		})
	}
}
