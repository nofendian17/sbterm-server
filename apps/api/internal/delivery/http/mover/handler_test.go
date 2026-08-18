package mover

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/mocks"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

func TestMarketMoverHandlerMarketMover(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		setup       func(uc *mocks.MockMarketMoverUsecase)
		wantStatus  int
		wantLen     int
		wantSymbol  string
		wantErrCode string
		wantDetails map[string]string
	}{
		{
			name:  "returns market movers",
			query: "?mover_type=MOVER_TYPE_TOP_GAINER&filter_stocks=FILTER_STOCKS_TYPE_MAIN_BOARD",
			setup: func(uc *mocks.MockMarketMoverUsecase) {
				uc.EXPECT().GetMarketMover(gomock.Any(), "MOVER_TYPE_TOP_GAINER", []string{"FILTER_STOCKS_TYPE_MAIN_BOARD"}).Return([]domain.MarketMover{
					{Symbol: "VOKS", Name: "Voksel Electric Tbk.", Price: 270, ChangeValue: 70, ChangePercent: 35, Value: 4795797800, Volume: 183517, Frequency: 2743, NetForeignSell: 164914400, IEP: 250, IEV: 260, IEVAL: 255, IEPChangePrev: 75},
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
			wantSymbol: "VOKS",
		},
		{
			name:        "rejects invalid mover type",
			query:       "?mover_type=INVALID",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantDetails: map[string]string{"mover_type": "must be one of: MOVER_TYPE_TOP_GAINER MOVER_TYPE_TOP_LOSER MOVER_TYPE_TOP_VALUE MOVER_TYPE_TOP_VOLUME MOVER_TYPE_TOP_FREQUENCY MOVER_TYPE_NET_FOREIGN_BUY MOVER_TYPE_NET_FOREIGN_SELL MOVER_TYPE_IEVAL_TOP_GAINER"},
		},
		{
			name:        "rejects missing mover type",
			query:       "",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantDetails: map[string]string{"mover_type": "is required"},
		},
		{
			name:        "rejects invalid filter board",
			query:       "?mover_type=MOVER_TYPE_TOP_GAINER&filter_stocks=WRONG_BOARD",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "VALIDATION_ERROR",
			wantDetails: map[string]string{"filter_stocks[0]": "must be one of: FILTER_STOCKS_TYPE_MAIN_BOARD FILTER_STOCKS_TYPE_DEVELOPMENT_BOARD FILTER_STOCKS_TYPE_ACCELERATION_BOARD FILTER_STOCKS_TYPE_NEW_ECONOMY_BOARD FILTER_STOCKS_TYPE_SPECIAL_MONITORING_BOARD FILTER_STOCKS_TYPE_WARRANT_AND_RIGHT"},
		},
		{
			name:  "usecase error returns 500",
			query: "?mover_type=MOVER_TYPE_TOP_GAINER",
			setup: func(uc *mocks.MockMarketMoverUsecase) {
				uc.EXPECT().GetMarketMover(gomock.Any(), "MOVER_TYPE_TOP_GAINER", nil).Return(nil, errors.New("boom"))
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			uc := mocks.NewMockMarketMoverUsecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			h := NewMarketMoverHandler(uc, validator.New())
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/market-mover"+tt.query, nil)
			h.MarketMover(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var env struct {
				Success bool `json:"success"`
				Data    []struct {
					Symbol string `json:"symbol"`
					Name   string `json:"name"`
				} `json:"data"`
				Error *struct {
					Code    string            `json:"code"`
					Details map[string]string `json:"details"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))

			if tt.wantErrCode != "" {
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantErrCode, env.Error.Code)
				assert.Equal(t, tt.wantDetails, env.Error.Details)
				return
			}
			require.Len(t, env.Data, tt.wantLen)
			assert.Equal(t, tt.wantSymbol, env.Data[0].Symbol)
		})
	}
}
