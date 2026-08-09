package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

// MarketMoverRepository fetches market movers from the Stockbit API.
type MarketMoverRepository struct {
	client *stockbit.Client
}

func NewMarketMoverRepository(client *stockbit.Client) *MarketMoverRepository {
	return &MarketMoverRepository{client: client}
}

func (r *MarketMoverRepository) GetMarketMover(ctx context.Context, moverType string, filterStocks []string) ([]domain.MarketMover, error) {
	req := stockbit.MarketMoverRequest{}
	if moverType != "" {
		req.MoverType = stockbit.MoverType(moverType)
	}
	for _, f := range filterStocks {
		req.FilterStocks = append(req.FilterStocks, stockbit.FilterStocks(f))
	}

	resp, err := r.client.GetMarketMover(ctx, req)
	if err != nil {
		return nil, err
	}
	movers := make([]domain.MarketMover, 0, len(resp.Data.MoverList))
	for _, m := range resp.Data.MoverList {
		movers = append(movers, domain.MarketMover{
			Symbol:         m.StockDetail.Code,
			Name:           m.StockDetail.Name,
			Price:          m.Price,
			ChangeValue:    m.Change.Value,
			ChangePercent:  m.Change.Percentage,
			Value:          m.Value.Raw,
			Volume:         m.Volume.Raw,
			Frequency:      m.Frequency.Raw,
			NetForeignBuy:  m.NetForeignBuy.Raw,
			NetForeignSell: m.NetForeignSell.Raw,
			IEP:            m.IEPIEVDetail.IEP.Raw,
			IEV:            m.IEPIEVDetail.IEV.Raw,
			IEVAL:          m.IEPIEVDetail.IEVAL.Raw,
			IEPChangePrev:  m.IEPIEVDetail.IEPChangePrev.Raw,
		})
	}
	return movers, nil
}

var _ repository.MarketMoverRepository = (*MarketMoverRepository)(nil)
