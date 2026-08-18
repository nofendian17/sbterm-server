package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// MarketSessionRepository fetches the market session state from the Stockbit API.
type MarketSessionRepository struct {
	client *stockbit.Client
}

func NewMarketSessionRepository(client *stockbit.Client) *MarketSessionRepository {
	return &MarketSessionRepository{client: client}
}

func (r *MarketSessionRepository) GetMarketSession(ctx context.Context) (*domain.MarketSession, error) {
	resp, err := r.client.GetMarketSession(ctx)
	if err != nil {
		return nil, err
	}
	return &domain.MarketSession{
		Datetime: resp.Data.Datetime,
		FCA:      toSegment(resp.Data.Detail.FCA),
		Regular:  toSegment(resp.Data.Detail.Regular),
	}, nil
}

func toSegment(s stockbit.SessionInfo) domain.MarketSessionSegment {
	return domain.MarketSessionSegment{
		StateName:      s.StateName,
		IsLastSession:  s.IsLastSession,
		IsEndOfDay:     s.IsEndOfDay,
		StateStartTime: s.StateStartTime,
		StateEndTime:   s.StateEndTime,
		TimeLeft:       s.TimeLeft.Formatted,
	}
}

var _ repository.MarketSessionRepository = (*MarketSessionRepository)(nil)
