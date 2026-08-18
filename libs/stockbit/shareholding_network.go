package stockbit

import (
	"context"
	"net/url"
	"strconv"
)

const shareholdingNetworkPath = "/insider/shareholding/network"

// ShareholdingNetworkResponse is the shareholding network response: data is
// the network itself.
type ShareholdingNetworkResponse struct {
	Message string                  `json:"message"`
	Data    ShareholdingNetworkData `json:"data"`
}

type ShareholdingNetworkData struct {
	RootID     string                    `json:"root_id"`
	RootType   string                    `json:"root_type"`
	ReportDate string                    `json:"report_date"`
	Nodes      []ShareholdingNetworkNode `json:"nodes"`
	Edges      []ShareholdingNetworkEdge `json:"edges"`
}

type ShareholdingNetworkNode struct {
	ID         string                      `json:"id"`
	NodeType   string                      `json:"node_type"`
	Metadata   ShareholdingNetworkMetadata `json:"metadata"`
	MinDepth   int                         `json:"min_depth"`
	IsRendered bool                        `json:"is_rendered"`
}

type ShareholdingNetworkMetadata struct {
	Company  *ShareholdingNetworkCompany  `json:"company"`
	Investor *ShareholdingNetworkInvestor `json:"investor"`
}

type ShareholdingNetworkCompany struct {
	ID      int64  `json:"id"`
	Symbol  string `json:"symbol"`
	Name    string `json:"name"`
	IconURL string `json:"icon_url"`
}

type ShareholdingNetworkInvestor struct {
	ID                     int64                    `json:"id"`
	Name                   string                   `json:"name"`
	InvestorType           ShareholdingRawFormatted `json:"investor_type"`
	Location               ShareholdingRawFormatted `json:"location"`
	Nationality            ShareholdingRawFormatted `json:"nationality"`
	Domicile               ShareholdingRawFormatted `json:"domicile"`
	InvestorClassification ShareholdingRawFormatted `json:"investor_classification"`
}

type ShareholdingNetworkEdge struct {
	FromID       string                     `json:"from_id"`
	ToID         string                     `json:"to_id"`
	Shareholding ShareholdingNetworkHolding `json:"shareholding"`
	IsRendered   bool                       `json:"is_rendered"`
}

type ShareholdingNetworkHolding struct {
	Scripless   ShareholdingRawFormatted `json:"scripless"`
	Scrip       ShareholdingRawFormatted `json:"scrip"`
	TotalShares ShareholdingRawFormatted `json:"total_shares"`
	Percentage  ShareholdingPercent      `json:"percentage"`
}

// GetShareholdingNetwork returns the shareholding network for a root node. The
// access token is attached automatically.
func (c *Client) GetShareholdingNetwork(ctx context.Context, rootID, rootType string, maxDepth, maxEdgePerNode int) (*ShareholdingNetworkResponse, error) {
	q := url.Values{}
	q.Set("root_id", rootID)
	q.Set("root_type", rootType)
	if maxDepth > 0 {
		q.Set("max_depth", strconv.Itoa(maxDepth))
	}
	if maxEdgePerNode > 0 {
		q.Set("max_edge_per_node", strconv.Itoa(maxEdgePerNode))
	}
	var out ShareholdingNetworkResponse
	if err := c.Get(ctx, shareholdingNetworkPath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
