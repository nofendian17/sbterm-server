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
