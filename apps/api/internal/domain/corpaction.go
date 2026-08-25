package domain

import "encoding/json"

type CompanyCorpAction struct {
	ActionType string
	Rups       *RupsInfo
	RightIssue *RightIssueInfo
	StockSplit *StockSplitInfo
	Raw        json.RawMessage
}

type RupsInfo struct {
	CompanyID          string
	CompanySymbol      string
	CorpActionActive   bool
	CompanyName        string
	CompanyIconURL     string
	RupsCreated        string
	RupsDatahash       string
	RupsDate           string
	RupsID             string
	RupsTime           string
	RupsIqpAgenda      string
	RupsIqpID          string
	RupsIqpRecDt       string
	RupsIqpRemark      string
	RupsIqpResult      string
	RupsIqpRevisedDate string
	RupsIqpType        string
	RupsVenue          string
	RupsEligibleDate   string
}

type RightIssueInfo struct {
	CompanyID                    string
	CompanySymbol                string
	CorpActionActive             bool
	RightIssueCompanyID          string
	RightIssueAdjFactor          float64
	RightIssueFactor             string
	RightIssueCreated            string
	RightIssueCumdate            string
	RightIssueExdate             string
	RightIssueLastupdate         string
	RightIssueID                 string
	RightIssueIqpID              string
	RightIssueLock               int
	RightIssueNew                string
	RightIssueNewShare           float64
	RightIssueOld                string
	RightIssuePrice              int
	RightIssuePriceAdj           float64
	RightIssuePriceFactor        float64
	RightIssuePriceFormatted     string
	RightIssueRatio              string
	RightIssueRecdate            string
	RightIssueSubdate            string
	RightIssueTradingEnd         string
	RightIssueTradingStart       string
	RightIssueForeignPercentage  int
	RightIssueLocalPercentage    int
	RightIssueNumberOfSecurities int
	RightIssueTotal              int
	EventNote                    string
}

type StockSplitInfo struct {
	CompanyID            string
	CompanySymbol        string
	CorpActionActive     bool
	StockSplitCreated    string
	StockSplitCumdate    string
	StockSplitExdate     string
	StockSplitFactor     string
	StockSplitID         string
	StockSplitIqpID      string
	StockSplitLastupdate string
	StockSplitLock       int
	StockSplitNew        string
	StockSplitNewPrice   int
	StockSplitNewShare   int
	StockSplitOld        string
	StockSplitRatio      string
	StockSplitRecdate    string
	EventNote            string
}

// CorpActionCalendar groups a day's corporate action events by kind. Kinds
// whose upstream shape has not been observed are kept verbatim as raw JSON;
// Today lists the company ids that have any event on the date.
type CorpActionCalendar struct {
	Bonus         []json.RawMessage
	Dividend      []DividendInfo
	Economic      []json.RawMessage
	Ipo           []json.RawMessage
	Pubex         []json.RawMessage
	RightIssue    []RightIssueInfo
	Rups          []RupsInfo
	StockReverse  []json.RawMessage
	StockSplit    []StockSplitInfo
	Tender        []TenderInfo
	Warrant       []WarrantInfo
	StockDividend []json.RawMessage
	Today         []string
}

type DividendInfo struct {
	CompanyID              string
	CompanySymbol          string
	CorpActionActive       bool
	DividendCreated        string
	DividendCumdate        string
	DividendDatahash       string
	DividendExdate         string
	DividendID             string
	DividendIqpID          string
	DividendLastupdate     string
	DividendLock           int
	DividendPaydate        string
	DividendRecdate        string
	DividendValue          string
	Lastprice              string
	EventNote              string
	DividendValueFormatted string
	LastpriceFormatted     string
	DividendCurrency       string
	DividendFiscalYear     int
	DividendValueAdjusted  int
}

type TenderInfo struct {
	CompanyID            string
	CompanyName          string
	CompanySymbol        string
	CorpActionActive     bool
	TenderCreated        string
	TenderDatahash       string
	TenderEnd            string
	TenderID             string
	TenderPaydate        string
	TenderPercentage     string
	TenderPrice          string
	TenderShares         string
	TenderStart          string
	EventNote            string
	TenderPriceFormatted string
}

// WarrantInfo mirrors the upstream payload, including its "wrant_" field
// prefix spelling.
type WarrantInfo struct {
	CompanyID               string
	CompanySymbol           string
	CorpActionActive        bool
	WrantExcEnd             string
	WrantExcFrom            string
	WrantExcPrice           string
	WrantID                 string
	WrantIqpID              string
	WrantLastupdate         string
	WrantSerie              string
	WrantTotal              string
	WrantTradingEnd         string
	WrantTradingFrom        string
	EventNote               string
	WrantExcPriceFormatted  string
	WrantForeignPercentage  int
	WrantLocalPercentage    int
	WrantNumberOfSecurities int
	WrantCompanyID          string
}
