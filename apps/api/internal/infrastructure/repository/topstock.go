package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// TopStockRepository fetches top-stock leaderboards from the Stockbit API.
type TopStockRepository struct {
	client *stockbit.Client
}

func NewTopStockRepository(client *stockbit.Client) *TopStockRepository {
	return &TopStockRepository{client: client}
}

func (r *TopStockRepository) GetTopStock(ctx context.Context, start, end, investorType, marketType, valueType string, page int) (*domain.TopStockData, error) {
	resp, err := r.client.GetTopStock(ctx, start, end, investorType, marketType, valueType, page)
	if err != nil {
		return nil, err
	}
	ri := resp.Data.ResponseInfo
	di := resp.Data.DisplayOption
	return &domain.TopStockData{
		TopBuy:  toDomainTopStockItems(resp.Data.TopBuy),
		TopSell: toDomainTopStockItems(resp.Data.TopSell),
		Total:   toDomainTopStockItems(resp.Data.Total),
		ResponseInfo: domain.TopStockResponseInfo{
			Page:           ri.Page,
			Limit:          ri.Limit,
			MaxDayDuration: ri.MaxDayDuration,
			StartDate:      ri.StartDate,
			EndDate:        ri.EndDate,
			ValueType:      ri.ValueType,
		},
		DisplayOption: domain.TopStockDisplayOption{
			BannerMessage:      di.BannerMessage,
			ForeignValueColumn: di.ForeignValueColumn,
			EnabledValueType: domain.TopStockEnabledValueType{
				Gross: di.EnabledValueType.Gross,
				Net:   di.EnabledValueType.Net,
				Total: di.EnabledValueType.Total,
			},
		},
	}, nil
}

func toDomainTopStockItems(items []stockbit.TopStockItem) []domain.TopStockItem {
	out := make([]domain.TopStockItem, 0, len(items))
	for _, i := range items {
		out = append(out, domain.TopStockItem{
			Rank:    i.Rank,
			Code:    i.Code,
			IconURL: i.IconURL,
			Value: domain.RawFormatted{
				Raw:       i.Value.Raw,
				Formatted: i.Value.Formatted,
			},
			Lot: domain.RawFormatted{
				Raw:       i.Lot.Raw,
				Formatted: i.Lot.Formatted,
			},
			Average: domain.RawFormatted{
				Raw:       i.Average.Raw,
				Formatted: i.Average.Formatted,
			},
			ForeignValue: domain.RawFormatted{
				Raw:       i.ForeignValue.Raw,
				Formatted: i.ForeignValue.Formatted,
			},
			Frequency: domain.RawFormatted{
				Raw:       i.Frequency.Raw,
				Formatted: i.Frequency.Formatted,
			},
		})
	}
	return out
}

var _ repository.TopStockRepository = (*TopStockRepository)(nil)
