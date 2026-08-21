package repository

import (
	"context"
	"errors"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// MarketDetectorRepository fetches market detector data from the Stockbit API.
type MarketDetectorRepository struct {
	client *stockbit.Client
}

func NewMarketDetectorRepository(client *stockbit.Client) *MarketDetectorRepository {
	return &MarketDetectorRepository{client: client}
}

func (r *MarketDetectorRepository) GetMarketDetector(ctx context.Context, symbol, from, to, transactionType, marketBoard, investorType string, limit int) (*domain.MarketDetectorData, error) {
	resp, err := r.client.GetMarketDetector(ctx, symbol, from, to, transactionType, marketBoard, investorType, limit)
	if err != nil {
		// Translate the client's typed status error into a domain error so the
		// delivery handler can map upstream 4xx responses (e.g. 400 for an
		// invalid date range) to a client-facing status.
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg, RetryAfter: se.RetryAfter}
		}
		return nil, err
	}
	return &domain.MarketDetectorData{
		BandarDetector: domain.BandarDetector{
			Average:             resp.Data.BandarDetector.Average,
			Avg:                 domain.BandarAccdist{Accdist: resp.Data.BandarDetector.Avg.Accdist, Amount: resp.Data.BandarDetector.Avg.Amount, Percent: resp.Data.BandarDetector.Avg.Percent, Vol: resp.Data.BandarDetector.Avg.Vol},
			Avg5:                domain.BandarAccdist{Accdist: resp.Data.BandarDetector.Avg5.Accdist, Amount: resp.Data.BandarDetector.Avg5.Amount, Percent: resp.Data.BandarDetector.Avg5.Percent, Vol: resp.Data.BandarDetector.Avg5.Vol},
			BrokerAccdist:       resp.Data.BandarDetector.BrokerAccdist,
			NumberBrokerBuysell: resp.Data.BandarDetector.NumberBrokerBuysell,
			Top1:                domain.BandarAccdist{Accdist: resp.Data.BandarDetector.Top1.Accdist, Amount: resp.Data.BandarDetector.Top1.Amount, Percent: resp.Data.BandarDetector.Top1.Percent, Vol: resp.Data.BandarDetector.Top1.Vol},
			Top3:                domain.BandarAccdist{Accdist: resp.Data.BandarDetector.Top3.Accdist, Amount: resp.Data.BandarDetector.Top3.Amount, Percent: resp.Data.BandarDetector.Top3.Percent, Vol: resp.Data.BandarDetector.Top3.Vol},
			Top5:                domain.BandarAccdist{Accdist: resp.Data.BandarDetector.Top5.Accdist, Amount: resp.Data.BandarDetector.Top5.Amount, Percent: resp.Data.BandarDetector.Top5.Percent, Vol: resp.Data.BandarDetector.Top5.Vol},
			Top10:               domain.BandarAccdist{Accdist: resp.Data.BandarDetector.Top10.Accdist, Amount: resp.Data.BandarDetector.Top10.Amount, Percent: resp.Data.BandarDetector.Top10.Percent, Vol: resp.Data.BandarDetector.Top10.Vol},
			TotalBuyer:          resp.Data.BandarDetector.TotalBuyer,
			TotalSeller:         resp.Data.BandarDetector.TotalSeller,
			Value:               resp.Data.BandarDetector.Value,
			Volume:              resp.Data.BandarDetector.Volume,
		},
		BrokerSummary: domain.BrokerSummary{
			BrokersBuy:  toDomainBrokerBuys(resp.Data.BrokerSummary.BrokersBuy),
			BrokersSell: toDomainBrokerSells(resp.Data.BrokerSummary.BrokersSell),
			Symbol:      resp.Data.BrokerSummary.Symbol,
		},
		From: resp.Data.From,
		To:   resp.Data.To,
	}, nil
}

func toDomainBrokerBuys(items []stockbit.BrokerBuy) []domain.BrokerBuy {
	out := make([]domain.BrokerBuy, 0, len(items))
	for _, b := range items {
		out = append(out, domain.BrokerBuy{
			Blot:             b.Blot,
			Blotv:            b.Blotv,
			Bval:             b.Bval,
			Bvalv:            b.Bvalv,
			NetbsBrokerCode:  b.NetbsBrokerCode,
			NetbsBuyAvgPrice: b.NetbsBuyAvgPrice,
			NetbsDate:        b.NetbsDate,
			NetbsStockCode:   b.NetbsStockCode,
			Type:             b.Type,
			Freq:             b.Freq,
		})
	}
	return out
}

func toDomainBrokerSells(items []stockbit.BrokerSell) []domain.BrokerSell {
	out := make([]domain.BrokerSell, 0, len(items))
	for _, b := range items {
		out = append(out, domain.BrokerSell{
			NetbsBrokerCode:   b.NetbsBrokerCode,
			NetbsDate:         b.NetbsDate,
			NetbsSellAvgPrice: b.NetbsSellAvgPrice,
			NetbsStockCode:    b.NetbsStockCode,
			Slot:              b.Slot,
			Slotv:             b.Slotv,
			Sval:              b.Sval,
			Svalv:             b.Svalv,
			Type:              b.Type,
			Freq:              b.Freq,
		})
	}
	return out
}

var _ repository.MarketDetectorRepository = (*MarketDetectorRepository)(nil)
