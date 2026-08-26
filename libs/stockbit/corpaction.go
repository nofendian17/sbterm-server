package stockbit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

const corpActionPath = "/corpaction/%s"

const corpActionCalendarPath = "/corpaction"

// CorpActionResponse is the corp action response: data is the action list.
type CorpActionResponse struct {
	Message string              `json:"message"`
	Data    []CompanyCorpAction `json:"data"`
}

// CorpActionCalendarResponse is the date-based corp action response: data
// groups events by action kind.
type CorpActionCalendarResponse struct {
	Message string                 `json:"message"`
	Data    CorpActionCalendarData `json:"data"`
}

// CorpActionCalendarData groups a day's corporate action events by kind. Kinds
// whose upstream shape has not been observed are kept verbatim as raw JSON so
// nothing is dropped; Today lists the company ids that have any event.
type CorpActionCalendarData struct {
	Bonus         []json.RawMessage `json:"bonus"`
	Dividend      []DividendInfo    `json:"dividend"`
	Economic      []json.RawMessage `json:"economic"`
	Ipo           []json.RawMessage `json:"ipo"`
	Pubex         []json.RawMessage `json:"pubex"`
	RightIssue    []RightIssueInfo  `json:"rightissue"`
	Rups          []RupsInfo        `json:"rups"`
	StockReverse  []json.RawMessage `json:"stock_reverse"`
	StockSplit    []StockSplitInfo  `json:"stocksplit"`
	Tender        []TenderInfo      `json:"tender"`
	Warrant       []WarrantInfo     `json:"warrant"`
	StockDividend []json.RawMessage `json:"stock_dividend"`
	// Today is the upstream "today" marker. Its shape is unstable across
	// upstream releases: it has appeared both as an array of company ids
	// (["101","202"]) and as a single date string ("2026-08-25"). CorpActionToday
	// decodes either shape into []string so an upstream schema drift cannot
	// 500 the calendar endpoint.
	Today CorpActionToday `json:"today"`
}

// CorpActionToday is the "today" marker on a corp action calendar. Upstream has
// returned it both as a JSON array of company ids and as a bare date string; this
// type normalizes both into a []string so decoding never fails on an upstream
// shape change.
type CorpActionToday []string

// UnmarshalJSON tolerates the upstream array-of-ids shape and the scalar
// date-string shape, normalizing both to a string slice. An unexpected shape is
// dropped (leaving the slice empty) rather than failing the whole decode.
func (t *CorpActionToday) UnmarshalJSON(b []byte) error {
	*t = nil
	var ids []string
	if err := json.Unmarshal(b, &ids); err == nil {
		*t = CorpActionToday(ids)
		return nil
	}
	var single string
	if err := json.Unmarshal(b, &single); err == nil {
		if single != "" {
			*t = CorpActionToday{single}
		}
		return nil
	}
	return nil
}

// MarshalJSON emits the normalized slice, keeping the API response contract (an
// array under "today") stable regardless of what upstream sent.
func (t CorpActionToday) MarshalJSON() ([]byte, error) {
	if t == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(t))
}

type DividendInfo struct {
	CompanyID              string `json:"company_id"`
	CompanySymbol          string `json:"company_symbol"`
	CorpActionActive       bool   `json:"corp_action_active"`
	DividendCreated        string `json:"dividend_created"`
	DividendCumdate        string `json:"dividend_cumdate"`
	DividendDatahash       string `json:"dividend_datahash"`
	DividendExdate         string `json:"dividend_exdate"`
	DividendID             string `json:"dividend_id"`
	DividendIqpID          string `json:"dividend_iqp_id"`
	DividendLastupdate     string `json:"dividend_lastupdate"`
	DividendLock           int    `json:"dividend_lock"`
	DividendPaydate        string `json:"dividend_paydate"`
	DividendRecdate        string `json:"dividend_recdate"`
	DividendValue          string `json:"dividend_value"`
	Lastprice              string `json:"lastprice"`
	EventNote              string `json:"event_note"`
	DividendValueFormatted string `json:"dividend_value_formatted"`
	LastpriceFormatted     string `json:"lastprice_formatted"`
	DividendCurrency       string `json:"dividend_currency"`
	DividendFiscalYear     int    `json:"dividend_fiscal_year"`
	DividendValueAdjusted  int    `json:"dividend_value_adjusted"`
}

type TenderInfo struct {
	CompanyID            string `json:"company_id"`
	CompanyName          string `json:"company_name"`
	CompanySymbol        string `json:"company_symbol"`
	CorpActionActive     bool   `json:"corp_action_active"`
	TenderCreated        string `json:"tender_created"`
	TenderDatahash       string `json:"tender_datahash"`
	TenderEnd            string `json:"tender_end"`
	TenderID             string `json:"tender_id"`
	TenderPaydate        string `json:"tender_paydate"`
	TenderPercentage     string `json:"tender_percentage"`
	TenderPrice          string `json:"tender_price"`
	TenderShares         string `json:"tender_shares"`
	TenderStart          string `json:"tender_start"`
	EventNote            string `json:"event_note"`
	TenderPriceFormatted string `json:"tender_price_formatted"`
}

// WarrantInfo mirrors the upstream payload, including its "wrant_" field
// prefix spelling.
type WarrantInfo struct {
	CompanyID               string `json:"company_id"`
	CompanySymbol           string `json:"company_symbol"`
	CorpActionActive        bool   `json:"corp_action_active"`
	WrantExcEnd             string `json:"wrant_exc_end"`
	WrantExcFrom            string `json:"wrant_exc_from"`
	WrantExcPrice           string `json:"wrant_exc_price"`
	WrantID                 string `json:"wrant_id"`
	WrantIqpID              string `json:"wrant_iqp_id"`
	WrantLastupdate         string `json:"wrant_lastupdate"`
	WrantSerie              string `json:"wrant_serie"`
	WrantTotal              string `json:"wrant_total"`
	WrantTradingEnd         string `json:"wrant_trading_end"`
	WrantTradingFrom        string `json:"wrant_trading_from"`
	EventNote               string `json:"event_note"`
	WrantExcPriceFormatted  string `json:"wrant_exc_price_formatted"`
	WrantForeignPercentage  int    `json:"wrant_foreign_percentage"`
	WrantLocalPercentage    int    `json:"wrant_local_percentage"`
	WrantNumberOfSecurities int    `json:"wrant_number_of_securities"`
	WrantCompanyID          string `json:"wrant_company_id"`
}

// CompanyCorpAction is one corp action entry: the common action_type plus a typed
// payload for the matching variant. Unknown variants are kept verbatim in Raw.
type CompanyCorpAction struct {
	ActionType string
	Rups       *RupsInfo
	RightIssue *RightIssueInfo
	StockSplit *StockSplitInfo
	Raw        json.RawMessage
}

type RupsInfo struct {
	CompanyID          string `json:"company_id"`
	CompanySymbol      string `json:"company_symbol"`
	CorpActionActive   bool   `json:"corp_action_active"`
	CompanyName        string `json:"company_name"`
	CompanyIconURL     string `json:"company_icon_url"`
	RupsCreated        string `json:"rups_created"`
	RupsDatahash       string `json:"rups_datahash"`
	RupsDate           string `json:"rups_date"`
	RupsID             string `json:"rups_id"`
	RupsTime           string `json:"rups_time"`
	RupsIqpAgenda      string `json:"rups_iqp_agenda"`
	RupsIqpID          string `json:"rups_iqp_id"`
	RupsIqpRecDt       string `json:"rups_iqp_rec_dt"`
	RupsIqpRemark      string `json:"rups_iqp_remark"`
	RupsIqpResult      string `json:"rups_iqp_result"`
	RupsIqpRevisedDate string `json:"rups_iqp_revised_date"`
	RupsIqpType        string `json:"rups_iqp_type"`
	RupsVenue          string `json:"rups_venue"`
	RupsEligibleDate   string `json:"rups_eligible_date"`
}

type RightIssueInfo struct {
	CompanyID                    string  `json:"company_id"`
	CompanySymbol                string  `json:"company_symbol"`
	CorpActionActive             bool    `json:"corp_action_active"`
	RightIssueCompanyID          string  `json:"rightissue_company_id"`
	RightIssueAdjFactor          float64 `json:"rightissue_adj_factor"`
	RightIssueFactor             string  `json:"rightissue_factor"`
	RightIssueCreated            string  `json:"rightissue_created"`
	RightIssueCumdate            string  `json:"rightissue_cumdate"`
	RightIssueExdate             string  `json:"rightissue_exdate"`
	RightIssueLastupdate         string  `json:"rightissue_lastupdate"`
	RightIssueID                 string  `json:"rightissue_id"`
	RightIssueIqpID              string  `json:"rightissue_iqp_id"`
	RightIssueLock               int     `json:"rightissue_lock"`
	RightIssueNew                string  `json:"rightissue_new"`
	RightIssueNewShare           float64 `json:"rightissue_new_share"`
	RightIssueOld                string  `json:"rightissue_old"`
	RightIssuePrice              int     `json:"rightissue_price"`
	RightIssuePriceAdj           float64 `json:"rightissue_price_adj"`
	RightIssuePriceFactor        float64 `json:"rightissue_price_factor"`
	RightIssuePriceFormatted     string  `json:"rightissue_price_formatted"`
	RightIssueRatio              string  `json:"rightissue_ratio"`
	RightIssueRecdate            string  `json:"rightissue_recdate"`
	RightIssueSubdate            string  `json:"rightissue_subdate"`
	RightIssueTradingEnd         string  `json:"rightissue_trading_end"`
	RightIssueTradingStart       string  `json:"rightissue_trading_start"`
	RightIssueForeignPercentage  int     `json:"rightissue_foreign_percentage"`
	RightIssueLocalPercentage    int     `json:"rightissue_local_percentage"`
	RightIssueNumberOfSecurities int     `json:"rightissue_number_of_securities"`
	RightIssueTotal              int     `json:"rightissue_total"`
	EventNote                    string  `json:"event_note"`
}

type StockSplitInfo struct {
	CompanyID            string `json:"company_id"`
	CompanySymbol        string `json:"company_symbol"`
	CorpActionActive     bool   `json:"corp_action_active"`
	StockSplitCreated    string `json:"stocksplit_created"`
	StockSplitCumdate    string `json:"stocksplit_cumdate"`
	StockSplitExdate     string `json:"stocksplit_exdate"`
	StockSplitFactor     string `json:"stocksplit_factor"`
	StockSplitID         string `json:"stocksplit_id"`
	StockSplitIqpID      string `json:"stocksplit_iqp_id"`
	StockSplitLastupdate string `json:"stocksplit_lastupdate"`
	StockSplitLock       int    `json:"stocksplit_lock"`
	StockSplitNew        string `json:"stocksplit_new"`
	StockSplitNewPrice   int    `json:"stocksplit_new_price"`
	StockSplitNewShare   int    `json:"stocksplit_new_share"`
	StockSplitOld        string `json:"stocksplit_old"`
	StockSplitRatio      string `json:"stocksplit_ratio"`
	StockSplitRecdate    string `json:"stocksplit_recdate"`
	EventNote            string `json:"event_note"`
}

// UnmarshalJSON dispatches action_info on action_type and decodes the matching
// variant; unknown variants are preserved verbatim in Raw.
func (a *CompanyCorpAction) UnmarshalJSON(b []byte) error {
	var aux struct {
		ActionType string          `json:"action_type"`
		ActionInfo json.RawMessage `json:"action_info"`
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	a.ActionType = aux.ActionType
	if len(aux.ActionInfo) == 0 {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(aux.ActionInfo, &m); err != nil {
		return err
	}
	raw := m[aux.ActionType]
	switch aux.ActionType {
	case "rups":
		v := new(RupsInfo)
		if err := json.Unmarshal(raw, &v); err != nil {
			return err
		}
		a.Rups = v
	case "rightissue":
		v := new(RightIssueInfo)
		if err := json.Unmarshal(raw, &v); err != nil {
			return err
		}
		a.RightIssue = v
	case "stocksplit":
		v := new(StockSplitInfo)
		if err := json.Unmarshal(raw, &v); err != nil {
			return err
		}
		a.StockSplit = v
	default:
		a.Raw = raw
	}
	return nil
}

// MarshalJSON reassembles the upstream shape: {"action_type":t,
// "action_info":{t:{...}}}.
func (a CompanyCorpAction) MarshalJSON() ([]byte, error) {
	info := map[string]any{}
	switch {
	case a.Rups != nil:
		info["rups"] = a.Rups
	case a.RightIssue != nil:
		info["rightissue"] = a.RightIssue
	case a.StockSplit != nil:
		info["stocksplit"] = a.StockSplit
	case a.Raw != nil:
		info[a.ActionType] = a.Raw
	}
	return json.Marshal(struct {
		ActionType string `json:"action_type"`
		ActionInfo any    `json:"action_info"`
	}{a.ActionType, info})
}

// GetCorpActions returns the corp actions for a symbol. The access token is
// attached automatically.
func (c *Client) GetCorpActions(ctx context.Context, symbol string, limit int) (*CorpActionResponse, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var out CorpActionResponse
	if err := c.Get(ctx, fmt.Sprintf(corpActionPath, symbol), q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCorpActionCalendar returns the corporate action events for a date
// (YYYY-MM-DD). The access token is attached automatically.
func (c *Client) GetCorpActionCalendar(ctx context.Context, date string) (*CorpActionCalendarResponse, error) {
	q := url.Values{"date": []string{date}}
	var out CorpActionCalendarResponse
	if err := c.Get(ctx, corpActionCalendarPath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
