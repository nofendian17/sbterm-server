package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// BrokerTopRepository fetches the top broker ranking from the Stockbit API.
type BrokerTopRepository struct {
	client *stockbit.Client
}

func NewBrokerTopRepository(client *stockbit.Client) *BrokerTopRepository {
	return &BrokerTopRepository{client: client}
}

func (r *BrokerTopRepository) GetBrokerTop(ctx context.Context, sort, order, period, marketType string, eodOnly bool) (*domain.BrokerTopData, error) {
	resp, err := r.client.GetBrokerTop(ctx, sort, order, period, marketType, eodOnly)
	if err != nil {
		return nil, err
	}
	d := resp.Data
	out := &domain.BrokerTopData{
		Date: domain.BrokerTopDate{From: d.Date.From, To: d.Date.To, Idx: d.Date.Idx},
		List: make([]domain.BrokerTopItem, 0, len(d.List)),
	}
	for _, it := range d.List {
		out.List = append(out.List, domain.BrokerTopItem{
			Code:           it.Code,
			Name:           it.Name,
			InvestorType:   it.InvestorType,
			TotalValue:     it.TotalValue,
			NetValue:       it.NetValue,
			BuyValue:       it.BuyValue,
			SellValue:      it.SellValue,
			TotalVolume:    it.TotalVolume,
			TotalFrequency: it.TotalFrequency,
			Group:          it.Group,
		})
	}
	return out, nil
}

var _ repository.BrokerTopRepository = (*BrokerTopRepository)(nil)
