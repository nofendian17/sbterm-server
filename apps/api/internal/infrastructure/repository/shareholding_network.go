package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// ShareholdingNetworkRepository fetches a shareholding network from the
// Stockbit API.
type ShareholdingNetworkRepository struct {
	client *stockbit.Client
}

func NewShareholdingNetworkRepository(client *stockbit.Client) *ShareholdingNetworkRepository {
	return &ShareholdingNetworkRepository{client: client}
}

func (r *ShareholdingNetworkRepository) GetShareholdingNetwork(ctx context.Context, rootID, rootType string, maxDepth, maxEdgePerNode int) (*domain.ShareholdingNetwork, error) {
	resp, err := r.client.GetShareholdingNetwork(ctx, rootID, rootType, maxDepth, maxEdgePerNode)
	if err != nil {
		return nil, err
	}
	out := &domain.ShareholdingNetwork{
		RootID:     resp.Data.RootID,
		RootType:   resp.Data.RootType,
		ReportDate: resp.Data.ReportDate,
		Nodes:      make([]domain.ShareholdingNetworkNode, 0, len(resp.Data.Nodes)),
		Edges:      make([]domain.ShareholdingNetworkEdge, 0, len(resp.Data.Edges)),
	}
	for _, n := range resp.Data.Nodes {
		nd := domain.ShareholdingNetworkNode{
			ID:         n.ID,
			NodeType:   n.NodeType,
			MinDepth:   n.MinDepth,
			IsRendered: n.IsRendered,
		}
		if n.Metadata.Company != nil {
			nd.Metadata.Company = &domain.ShareholdingNetworkCompany{
				ID:      n.Metadata.Company.ID,
				Symbol:  n.Metadata.Company.Symbol,
				Name:    n.Metadata.Company.Name,
				IconURL: n.Metadata.Company.IconURL,
			}
		}
		if n.Metadata.Investor != nil {
			nd.Metadata.Investor = &domain.ShareholdingNetworkInvestor{
				ID:                     n.Metadata.Investor.ID,
				Name:                   n.Metadata.Investor.Name,
				InvestorType:           toRawFormattedDomain(n.Metadata.Investor.InvestorType),
				Location:               toRawFormattedDomain(n.Metadata.Investor.Location),
				Nationality:            toRawFormattedDomain(n.Metadata.Investor.Nationality),
				Domicile:               toRawFormattedDomain(n.Metadata.Investor.Domicile),
				InvestorClassification: toRawFormattedDomain(n.Metadata.Investor.InvestorClassification),
			}
		}
		out.Nodes = append(out.Nodes, nd)
	}
	for _, e := range resp.Data.Edges {
		out.Edges = append(out.Edges, domain.ShareholdingNetworkEdge{
			FromID: e.FromID,
			ToID:   e.ToID,
			Shareholding: domain.ShareholdingNetworkHolding{
				Scripless:   toRawFormattedDomain(e.Shareholding.Scripless),
				Scrip:       toRawFormattedDomain(e.Shareholding.Scrip),
				TotalShares: toRawFormattedDomain(e.Shareholding.TotalShares),
				Percentage:  domain.ShareholdingPercent{Raw: e.Shareholding.Percentage.Raw, Formatted: e.Shareholding.Percentage.Formatted},
			},
			IsRendered: e.IsRendered,
		})
	}
	return out, nil
}

func toRawFormattedDomain(in stockbit.ShareholdingRawFormatted) domain.ShareholdingRawFormatted {
	return domain.ShareholdingRawFormatted{Raw: in.Raw, Formatted: in.Formatted}
}

var _ repository.ShareholdingNetworkRepository = (*ShareholdingNetworkRepository)(nil)
