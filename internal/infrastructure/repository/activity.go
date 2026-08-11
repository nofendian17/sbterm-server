package repository

import (
	"context"
	"errors"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

// ActivityRepository fetches broker activity chart and transaction data from the
// Stockbit API.
type ActivityRepository struct {
	client *stockbit.Client
}

func NewActivityRepository(client *stockbit.Client) *ActivityRepository {
	return &ActivityRepository{client: client}
}

func (r *ActivityRepository) GetActivityChart(ctx context.Context, symbols, brokersCode []string, from, to, period, investorType, marketBoard string) (*domain.ActivityChartData, error) {
	resp, err := r.client.GetActivityChart(ctx, symbols, brokersCode, from, to, period, investorType, marketBoard)
	if err != nil {
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg}
		}
		return nil, err
	}
	d := resp.Data
	out := &domain.ActivityChartData{
		From:            d.From,
		To:              d.To,
		DataLastUpdated: d.DataLastUpdated,
		ChartData:       make([]domain.ActivityChartGroup, 0, len(d.ChartData)),
		DateSessionInfo: d.DateSessionInfo,
		BrokerCode:      d.BrokerCode,
		BrokerName:      d.BrokerName,
	}
	for _, g := range d.ChartData {
		group := domain.ActivityChartGroup{
			Type:    g.Type,
			Symbols: g.Symbols,
			Charts:  make([]domain.ActivityChartSeries, 0, len(g.Charts)),
		}
		for _, ch := range g.Charts {
			series := domain.ActivityChartSeries{
				Symbol: ch.Symbol,
				Chart:  make([]domain.ActivityChartPoint, 0, len(ch.Chart)),
			}
			for _, p := range ch.Chart {
				series.Chart = append(series.Chart, domain.ActivityChartPoint{
					Date:          p.Date,
					Time:          p.Time,
					Value:         domain.RawFormatted{Raw: p.Value.Raw, Formatted: p.Value.Formatted},
					DatetimeLabel: p.DatetimeLabel,
				})
			}
			group.Charts = append(group.Charts, series)
		}
		out.ChartData = append(out.ChartData, group)
	}
	return out, nil
}

func (r *ActivityRepository) GetActivity(ctx context.Context, brokerCode []string, transactionType, investorType, marketBoard string, limit, page int, from, to, netValPeriod string) (*domain.ActivityData, error) {
	resp, err := r.client.GetActivity(ctx, brokerCode, transactionType, investorType, marketBoard, limit, page, from, to, netValPeriod)
	if err != nil {
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg}
		}
		return nil, err
	}
	d := resp.Data
	ba := d.BrokerActivityTransaction
	out := &domain.ActivityData{
		BrokerActivityTransaction: domain.BrokerActivityTransaction{
			BrokersBuy:  toBrokerActivities(ba.BrokersBuy),
			BrokersSell: toBrokerActivities(ba.BrokersSell),
		},
		From:       d.From,
		To:         d.To,
		BrokerCode: d.BrokerCode,
		BrokerName: d.BrokerName,
	}
	return out, nil
}

func toBrokerActivities(in []stockbit.ActivityBrokerActivity) []domain.BrokerActivity {
	out := make([]domain.BrokerActivity, 0, len(in))
	for _, b := range in {
		out = append(out, domain.BrokerActivity{
			StockCode:    b.StockCode,
			BrokerCode:   b.BrokerCode,
			Type:         b.Type,
			Date:         b.Date,
			Value:        b.Value,
			Lot:          b.Lot,
			AveragePrice: b.AveragePrice,
			Frequency:    b.Frequency,
			CompanyDetail: domain.ActivityCompanyDetail{
				IconURL:    b.CompanyDetail.IconURL,
				CorpAction: domain.ActivityCorpAction{Active: b.CompanyDetail.CorpAction.Active, Icon: b.CompanyDetail.CorpAction.Icon, Text: b.CompanyDetail.CorpAction.Text},
				Notation:   toActivityNotations(b.CompanyDetail.Notation),
			},
			NetValueTrend: toNetValueTrend(b.NetValueTrend),
		})
	}
	return out
}

func toActivityNotations(in []stockbit.ActivityNotation) []domain.ActivityNotation {
	out := make([]domain.ActivityNotation, 0, len(in))
	for _, n := range in {
		out = append(out, domain.ActivityNotation{
			NotationCode: n.NotationCode,
			NotationDesc: n.NotationDesc,
			IconURL:      domain.ActivityNotationIcon{LightMode: n.IconURL.LightMode, DarkMode: n.IconURL.DarkMode},
		})
	}
	return out
}

func toNetValueTrend(in []stockbit.ActivityNetValueTrend) []domain.ActivityNetValueTrend {
	out := make([]domain.ActivityNetValueTrend, 0, len(in))
	for _, n := range in {
		out = append(out, domain.ActivityNetValueTrend{
			Date:  n.Date,
			NVal:  n.NVal,
			NVol:  n.NVol,
			NFreq: n.NFreq,
		})
	}
	return out
}

var _ repository.ActivityRepository = (*ActivityRepository)(nil)
