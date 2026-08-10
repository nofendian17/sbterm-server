package usecase

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=indexsummary.go -destination=../mocks/mock_indexsummary_usecase.go -package=mocks -typed
type IndexSummaryUsecase interface {
	GetIndexSummary(ctx context.Context, symbol, from, to, interval string) (*domain.IndexSummaryData, error)
	// GetIndexChart returns the index summary combined with chartbit OHLC bars
	// for the same index in a single response.
	GetIndexChart(ctx context.Context, symbol, from, to, interval string) (*domain.IndexChartData, error)
}

// maxSessionLookbackDays bounds how far the last-session default searches
// backwards for a trading day with data (covers weekends, holidays and
// pre-market hours when today's session has no points yet).
const maxSessionLookbackDays = 30

const dateLayout = "2006-01-02"

type indexSummaryUsecase struct {
	repo      repository.IndexSummaryRepository
	chartRepo repository.ChartbitRepository
}

func NewIndexSummaryUsecase(repo repository.IndexSummaryRepository, chartRepo repository.ChartbitRepository) *indexSummaryUsecase {
	return &indexSummaryUsecase{repo: repo, chartRepo: chartRepo}
}

func (u *indexSummaryUsecase) GetIndexSummary(ctx context.Context, symbol, from, to, interval string) (*domain.IndexSummaryData, error) {
	if from == "" && to == "" {
		_, s, err := u.lastSession(ctx, symbol, interval)
		return s, err
	}
	return u.repo.GetIndexSummary(ctx, symbol, from, to, interval)
}

// GetIndexChart fetches the summary and the OHLC bars concurrently and combines
// them. from/to are dates (YYYY-MM-DD) in chronological order (from = earlier,
// to = later), which is the convention the summary endpoint requires. Chartbit
// daily pages backward — from = newer date, to = older one — so the range is
// swapped for the chart call. When from/to are both empty the most recent
// trading session with data is used.
func (u *indexSummaryUsecase) GetIndexChart(ctx context.Context, symbol, from, to, interval string) (*domain.IndexChartData, error) {
	if from == "" && to == "" {
		date, s, err := u.lastSession(ctx, symbol, interval)
		if err != nil {
			return nil, err
		}
		chart, err := u.chartRepo.GetChartPrice(ctx, symbol, "daily", date, date, 0)
		if err != nil {
			return nil, err
		}
		return &domain.IndexChartData{Summary: *s, Chart: *chart}, nil
	}

	g, ctx := errgroup.WithContext(ctx)

	var summary *domain.IndexSummaryData
	var chart *domain.ChartPriceData

	g.Go(func() error {
		s, err := u.repo.GetIndexSummary(ctx, symbol, from, to, interval)
		if err != nil {
			return err
		}
		summary = s
		return nil
	})

	g.Go(func() error {
		c, err := u.chartRepo.GetChartPrice(ctx, symbol, "daily", to, from, 0)
		if err != nil {
			return err
		}
		chart = c
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &domain.IndexChartData{Summary: *summary, Chart: *chart}, nil
}

// lastSession finds the most recent trading day with price data by probing
// backwards from today (WIB, the IDX trading timezone), returning that date and
// its summary. It backs the from/to-empty default so pre-market, weekend and
// holiday requests still return the latest available session. An empty prices
// slice marks a day without a session; an upstream error is propagated as-is.
func (u *indexSummaryUsecase) lastSession(ctx context.Context, symbol, interval string) (string, *domain.IndexSummaryData, error) {
	wib := time.FixedZone("WIB", 7*3600)
	now := time.Now().In(wib)
	for i := 0; i < maxSessionLookbackDays; i++ {
		date := now.AddDate(0, 0, -i).Format(dateLayout)
		s, err := u.repo.GetIndexSummary(ctx, symbol, date, date, interval)
		if err != nil {
			return "", nil, err
		}
		if len(s.Prices) > 0 {
			return date, s, nil
		}
	}
	return "", nil, fmt.Errorf("index summary: no trading session with data in the last %d days", maxSessionLookbackDays)
}
