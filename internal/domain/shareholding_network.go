package domain

type ShareholdingNetwork struct {
	RootID     string
	RootType   string
	ReportDate string
	Nodes      []ShareholdingNetworkNode
	Edges      []ShareholdingNetworkEdge
}

type ShareholdingNetworkNode struct {
	ID         string
	NodeType   string
	Metadata   ShareholdingNetworkMetadata
	MinDepth   int
	IsRendered bool
}

type ShareholdingNetworkMetadata struct {
	Company  *ShareholdingNetworkCompany
	Investor *ShareholdingNetworkInvestor
}

type ShareholdingNetworkCompany struct {
	ID      int64
	Symbol  string
	Name    string
	IconURL string
}

type ShareholdingNetworkInvestor struct {
	ID                     int64
	Name                   string
	InvestorType           ShareholdingRawFormatted
	Location               ShareholdingRawFormatted
	Nationality            ShareholdingRawFormatted
	Domicile               ShareholdingRawFormatted
	InvestorClassification ShareholdingRawFormatted
}

type ShareholdingNetworkEdge struct {
	FromID       string
	ToID         string
	Shareholding ShareholdingNetworkHolding
	IsRendered   bool
}

type ShareholdingNetworkHolding struct {
	Scripless   ShareholdingRawFormatted
	Scrip       ShareholdingRawFormatted
	TotalShares ShareholdingRawFormatted
	Percentage  ShareholdingPercent
}
