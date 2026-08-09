package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

// MajorHolderRepository fetches major holder movements from the Stockbit API.
type MajorHolderRepository struct {
	client *stockbit.Client
}

func NewMajorHolderRepository(client *stockbit.Client) *MajorHolderRepository {
	return &MajorHolderRepository{client: client}
}

func (r *MajorHolderRepository) GetMajorHolder(ctx context.Context, symbols, actionType, sourceType string, page, limit int) (*domain.MajorHolderData, error) {
	resp, err := r.client.GetMajorHolder(ctx, symbols, actionType, sourceType, page, limit)
	if err != nil {
		return nil, err
	}
	out := &domain.MajorHolderData{
		IsMore:   resp.Data.IsMore,
		Movement: make([]domain.MajorHolderMovement, 0, len(resp.Data.Movement)),
	}
	for _, m := range resp.Data.Movement {
		out.Movement = append(out.Movement, domain.MajorHolderMovement{
			ID:             m.ID,
			Name:           m.Name,
			Symbol:         m.Symbol,
			Date:           m.Date,
			Previous:       domain.MajorHolderValueChange{Value: m.Previous.Value, Percentage: m.Previous.Percentage, FormattedValue: m.Previous.FormattedValue},
			Current:        domain.MajorHolderValueChange{Value: m.Current.Value, Percentage: m.Current.Percentage, FormattedValue: m.Current.FormattedValue},
			Changes:        domain.MajorHolderValueChange{Value: m.Changes.Value, Percentage: m.Changes.Percentage, FormattedValue: m.Changes.FormattedValue},
			Marker:         m.Marker,
			IsPosted:       m.IsPosted,
			CMHID:          m.CMHID,
			Nationality:    m.Nationality,
			ActionType:     m.ActionType,
			DataSource:     domain.MajorHolderDataSource{Label: m.DataSource.Label, Type: m.DataSource.Type},
			PriceFormatted: m.PriceFormatted,
			BrokerDetail:   domain.MajorHolderBroker{Code: m.BrokerDetail.Code, Group: m.BrokerDetail.Group},
			Badges:         m.Badges,
		})
	}
	return out, nil
}

var _ repository.MajorHolderRepository = (*MajorHolderRepository)(nil)
