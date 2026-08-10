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

func TestIndexSummaryUsecaseGetIndexSummary(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{name: "returns index summary"},
		{name: "propagates repository error", repoErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			want := &domain.IndexSummaryData{
				XAxisOpt: "intraday",
				Prices:   []domain.IndexSummaryPrice{{FormattedDate: "2026-08-10 09:00:00", Value: "6442.65"}},
			}
			repo := mocks.NewMockIndexSummaryRepository(ctrl)
			repo.EXPECT().GetIndexSummary(gomock.Any(), "IHSG", "2026-08-10", "2026-08-10", "INTERVAL_CHART_MINUTELY").Return(want, tt.repoErr)

			uc := NewIndexSummaryUsecase(repo, mocks.NewMockChartbitRepository(ctrl))
			got, err := uc.GetIndexSummary(context.Background(), "IHSG", "2026-08-10", "2026-08-10", "INTERVAL_CHART_MINUTELY")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}

func TestIndexSummaryUsecaseGetIndexChart(t *testing.T) {
	tests := []struct {
		name       string
		summaryErr error
		chartErr   error
		wantErr    bool
	}{
		{name: "combines summary and chart"},
		{name: "propagates summary error", summaryErr: errors.New("boom"), wantErr: true},
		{name: "propagates chart error", chartErr: errors.New("boom"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			wantSummary := &domain.IndexSummaryData{
				XAxisOpt: "intraday",
				Prices:   []domain.IndexSummaryPrice{{FormattedDate: "2026-08-10 09:00:00", Value: "6442.65"}},
			}
			wantChart := &domain.ChartPriceData{
				Chartbit: []domain.ChartPrice{{Close: 6365.374, High: 6462.738}},
			}
			repo := mocks.NewMockIndexSummaryRepository(ctrl)
			repo.EXPECT().GetIndexSummary(gomock.Any(), "IHSG", "2026-08-10", "2026-08-10", "INTERVAL_CHART_MINUTELY").Return(wantSummary, tt.summaryErr)
			chartRepo := mocks.NewMockChartbitRepository(ctrl)
			chartRepo.EXPECT().GetChartPrice(gomock.Any(), "IHSG", "daily", "2026-08-10", "2026-08-10", 0).Return(wantChart, tt.chartErr)

			uc := NewIndexSummaryUsecase(repo, chartRepo)
			got, err := uc.GetIndexChart(context.Background(), "IHSG", "2026-08-10", "2026-08-10", "INTERVAL_CHART_MINUTELY")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, wantSummary.XAxisOpt, got.Summary.XAxisOpt)
			require.Len(t, got.Chart.Chartbit, 1)
			assert.Equal(t, float64(6365.374), got.Chart.Chartbit[0].Close)
		})
	}
}

func TestIndexSummaryUsecaseGetIndexChartSwapsRangeForChartbit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockIndexSummaryRepository(ctrl)
	// Summary receives the chronological range as-is: from = earlier, to = later.
	repo.EXPECT().GetIndexSummary(gomock.Any(), "IHSG", "2025-08-10", "2026-08-10", "").Return(&domain.IndexSummaryData{}, nil)
	chartRepo := mocks.NewMockChartbitRepository(ctrl)
	// Chartbit daily pages backward: the range is swapped for the chart call.
	chartRepo.EXPECT().GetChartPrice(gomock.Any(), "IHSG", "daily", "2026-08-10", "2025-08-10", 0).Return(&domain.ChartPriceData{}, nil)

	uc := NewIndexSummaryUsecase(repo, chartRepo)
	got, err := uc.GetIndexChart(context.Background(), "IHSG", "2025-08-10", "2026-08-10", "")
	require.NoError(t, err)
	assert.NotNil(t, got)
}

func TestIndexSummaryUsecaseGetIndexSummaryDefaultsToLastSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	want := &domain.IndexSummaryData{
		XAxisOpt: "intraday",
		Prices:   []domain.IndexSummaryPrice{{FormattedDate: "2026-08-10 09:00:00", Value: "6442.65"}},
	}
	repo := mocks.NewMockIndexSummaryRepository(ctrl)
	// First probe (today, pre-market) returns no prices; the next day has data.
	gomock.InOrder(
		repo.EXPECT().GetIndexSummary(gomock.Any(), "IHSG", gomock.Any(), gomock.Any(), "INTERVAL_CHART_MINUTELY").Return(&domain.IndexSummaryData{}, nil),
		repo.EXPECT().GetIndexSummary(gomock.Any(), "IHSG", gomock.Any(), gomock.Any(), "INTERVAL_CHART_MINUTELY").Return(want, nil),
	)

	uc := NewIndexSummaryUsecase(repo, mocks.NewMockChartbitRepository(ctrl))
	got, err := uc.GetIndexSummary(context.Background(), "IHSG", "", "", "INTERVAL_CHART_MINUTELY")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestIndexSummaryUsecaseGetIndexSummaryNoSessionInLookback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockIndexSummaryRepository(ctrl)
	repo.EXPECT().GetIndexSummary(gomock.Any(), "IHSG", gomock.Any(), gomock.Any(), gomock.Any()).Return(&domain.IndexSummaryData{}, nil).AnyTimes()

	uc := NewIndexSummaryUsecase(repo, mocks.NewMockChartbitRepository(ctrl))
	_, err := uc.GetIndexSummary(context.Background(), "IHSG", "", "", "")
	require.Error(t, err)
}

func TestIndexSummaryUsecaseGetIndexChartDefaultsToLastSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	wantSummary := &domain.IndexSummaryData{
		XAxisOpt: "intraday",
		Prices:   []domain.IndexSummaryPrice{{FormattedDate: "2026-08-10 09:00:00", Value: "6442.65"}},
	}
	wantChart := &domain.ChartPriceData{Chartbit: []domain.ChartPrice{{Close: 6365.374}}}
	repo := mocks.NewMockIndexSummaryRepository(ctrl)
	gomock.InOrder(
		repo.EXPECT().GetIndexSummary(gomock.Any(), "IHSG", gomock.Any(), gomock.Any(), "").Return(&domain.IndexSummaryData{}, nil),
		repo.EXPECT().GetIndexSummary(gomock.Any(), "IHSG", gomock.Any(), gomock.Any(), "").Return(wantSummary, nil),
	)
	chartRepo := mocks.NewMockChartbitRepository(ctrl)
	chartRepo.EXPECT().GetChartPrice(gomock.Any(), "IHSG", "daily", gomock.Any(), gomock.Any(), 0).Return(wantChart, nil)

	uc := NewIndexSummaryUsecase(repo, chartRepo)
	got, err := uc.GetIndexChart(context.Background(), "IHSG", "", "", "")
	require.NoError(t, err)
	assert.Equal(t, wantSummary.XAxisOpt, got.Summary.XAxisOpt)
	require.Len(t, got.Chart.Chartbit, 1)
	assert.Equal(t, float64(6365.374), got.Chart.Chartbit[0].Close)
}
