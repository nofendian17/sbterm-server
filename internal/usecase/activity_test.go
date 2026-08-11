package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/mocks"
)

func TestActivityUsecaseGetActivityChart(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns activity chart"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.ActivityChartData{
				ChartData: []domain.ActivityChartGroup{{
					Type:    "TYPE_CHART_VALUE",
					Symbols: []string{"BBRI"},
					Charts: []domain.ActivityChartSeries{{
						Symbol: "BBRI",
						Chart: []domain.ActivityChartPoint{{
							Date:          "2026-08-03",
							Time:          "00:00",
							Value:         domain.RawFormatted{Raw: "835", Formatted: "835"},
							DatetimeLabel: "03 Aug",
						}},
					}},
				}},
			}
			repo := mocks.NewMockActivityRepository(ctrl)
			repo.EXPECT().GetActivityChart(gomock.Any(), []string{"BBRI"}, []string{"DR"}, "2026-07-01", "2026-08-10", "", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL").Return(want, tt.repoErr)

			uc := NewActivityUsecase(repo)
			got, err := uc.GetActivityChart(context.Background(), []string{"BBRI"}, []string{"DR"}, "2026-07-01", "2026-08-10", "", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}

func TestActivityUsecaseGetActivity(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns activity"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.ActivityData{
				BrokerActivityTransaction: domain.BrokerActivityTransaction{
					BrokersBuy: []domain.BrokerActivity{{
						StockCode:    "BBRI",
						BrokerCode:   "AK",
						Type:         "BROKER_TYPE_LOCAL",
						Date:         "2026-08-10",
						Value:        8356500000,
						Lot:          313400,
						AveragePrice: 2666,
						Frequency:    1203,
						CompanyDetail: domain.ActivityCompanyDetail{
							IconURL:    "https://x/BBRI.png",
							CorpAction: domain.ActivityCorpAction{Active: false, Icon: "", Text: ""},
							Notation:   []domain.ActivityNotation{{NotationCode: "X", NotationDesc: "Suspensi", IconURL: domain.ActivityNotationIcon{LightMode: "https://x/l.png", DarkMode: "https://x/d.png"}}},
						},
						NetValueTrend: []domain.ActivityNetValueTrend{{
							Date:  "2026-08-10",
							NVal:  8356500000,
							NVol:  313400,
							NFreq: 1203,
						}},
					}},
					BrokersSell: []domain.BrokerActivity{},
				},
			}
			repo := mocks.NewMockActivityRepository(ctrl)
			repo.EXPECT().GetActivity(gomock.Any(), []string{"AK"}, "TRANSACTION_TYPE_GROSS", "INVESTOR_TYPE_ALL", "MARKET_TYPE_REGULER", 20, 1, "2026-07-14", "2026-07-31", "NET_VAL_PERIOD_7D").Return(want, tt.repoErr)

			uc := NewActivityUsecase(repo)
			got, err := uc.GetActivity(context.Background(), []string{"AK"}, "TRANSACTION_TYPE_GROSS", "INVESTOR_TYPE_ALL", "MARKET_TYPE_REGULER", 20, 1, "2026-07-14", "2026-07-31", "NET_VAL_PERIOD_7D")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
