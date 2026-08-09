package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

// PricePerformanceRepository fetches price performance from the Stockbit API.
type PricePerformanceRepository struct {
	client *stockbit.Client
}

func NewPricePerformanceRepository(client *stockbit.Client) *PricePerformanceRepository {
	return &PricePerformanceRepository{client: client}
}

func (r *PricePerformanceRepository) GetPricePerformance(ctx context.Context, symbol string) (*domain.PricePerformanceData, error) {
	resp, err := r.client.GetPricePerformance(ctx, symbol)
	if err != nil {
		return nil, err
	}
	out := &domain.PricePerformanceData{Prices: make([]domain.PricePerformance, 0, len(resp.Data.Prices))}
	for _, p := range resp.Data.Prices {
		out.Prices = append(out.Prices, domain.PricePerformance{
			Close:      domain.PriceRawFormatted{Raw: p.Close.Raw, Formatted: p.Close.Formatted},
			High:       domain.PriceRawFormatted{Raw: p.High.Raw, Formatted: p.High.Formatted},
			Low:        domain.PriceRawFormatted{Raw: p.Low.Raw, Formatted: p.Low.Formatted},
			Percentage: domain.PricePercent{Raw: p.Percentage.Raw, Formatted: p.Percentage.Formatted},
			Timeframe:  p.Timeframe,
		})
	}
	return out, nil
}

var _ repository.PricePerformanceRepository = (*PricePerformanceRepository)(nil)
