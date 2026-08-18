package corpaction

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

type CorpActionHandler struct {
	uc usecase.CorpActionUsecase
	v  validator.Validator
}

func NewCorpActionHandler(uc usecase.CorpActionUsecase, v validator.Validator) *CorpActionHandler {
	return &CorpActionHandler{uc: uc, v: v}
}

type corpActionRequest struct {
	Symbol string `json:"symbol" validate:"required"`
	Limit  int    `json:"limit" validate:"omitempty"`
}

// corpActionResponse mirrors the upstream shape: action_type plus the matching
// typed action_info payload.
type corpActionResponse struct {
	ActionType string          `json:"action_type"`
	ActionInfo json.RawMessage `json:"action_info"`
}

type corpActionInfoResponse struct {
	Rups       *rupsResponse       `json:"rups,omitempty"`
	RightIssue *rightIssueResponse `json:"rightissue,omitempty"`
	StockSplit *stockSplitResponse `json:"stocksplit,omitempty"`
	Raw        json.RawMessage     `json:"-"`
}

type rupsResponse struct {
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

type rightIssueResponse struct {
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

type stockSplitResponse struct {
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

func (h *CorpActionHandler) CorpActions(w http.ResponseWriter, r *http.Request) {
	req := corpActionRequest{Symbol: chi.URLParam(r, "symbol")}
	if v := r.URL.Query().Get("limit"); v != "" {
		req.Limit, _ = strconv.Atoi(v)
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate corp action params")
		return
	}

	actions, err := h.uc.GetCorpActions(r.Context(), req.Symbol, req.Limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get corp actions")
		return
	}
	response.OK(w, toResponses(actions))
}

func toResponses(in []domain.CompanyCorpAction) []corpActionResponse {
	res := make([]corpActionResponse, 0, len(in))
	for _, a := range in {
		res = append(res, corpActionResponse{
			ActionType: a.ActionType,
			ActionInfo: toActionInfo(a),
		})
	}
	return res
}

func toActionInfo(a domain.CompanyCorpAction) json.RawMessage {
	if a.Raw != nil {
		b, _ := json.Marshal(map[string]json.RawMessage{a.ActionType: a.Raw})
		return b
	}
	info := corpActionInfoResponse{}
	switch {
	case a.Rups != nil:
		info.Rups = &rupsResponse{
			CompanyID:          a.Rups.CompanyID,
			CompanySymbol:      a.Rups.CompanySymbol,
			CorpActionActive:   a.Rups.CorpActionActive,
			CompanyName:        a.Rups.CompanyName,
			CompanyIconURL:     a.Rups.CompanyIconURL,
			RupsCreated:        a.Rups.RupsCreated,
			RupsDatahash:       a.Rups.RupsDatahash,
			RupsDate:           a.Rups.RupsDate,
			RupsID:             a.Rups.RupsID,
			RupsTime:           a.Rups.RupsTime,
			RupsIqpAgenda:      a.Rups.RupsIqpAgenda,
			RupsIqpID:          a.Rups.RupsIqpID,
			RupsIqpRecDt:       a.Rups.RupsIqpRecDt,
			RupsIqpRemark:      a.Rups.RupsIqpRemark,
			RupsIqpResult:      a.Rups.RupsIqpResult,
			RupsIqpRevisedDate: a.Rups.RupsIqpRevisedDate,
			RupsIqpType:        a.Rups.RupsIqpType,
			RupsVenue:          a.Rups.RupsVenue,
			RupsEligibleDate:   a.Rups.RupsEligibleDate,
		}
	case a.RightIssue != nil:
		out := rightIssueResponse{
			CompanyID:                    a.RightIssue.CompanyID,
			CompanySymbol:                a.RightIssue.CompanySymbol,
			CorpActionActive:             a.RightIssue.CorpActionActive,
			RightIssueCompanyID:          a.RightIssue.RightIssueCompanyID,
			RightIssueAdjFactor:          a.RightIssue.RightIssueAdjFactor,
			RightIssueFactor:             a.RightIssue.RightIssueFactor,
			RightIssueCreated:            a.RightIssue.RightIssueCreated,
			RightIssueCumdate:            a.RightIssue.RightIssueCumdate,
			RightIssueExdate:             a.RightIssue.RightIssueExdate,
			RightIssueLastupdate:         a.RightIssue.RightIssueLastupdate,
			RightIssueID:                 a.RightIssue.RightIssueID,
			RightIssueIqpID:              a.RightIssue.RightIssueIqpID,
			RightIssueLock:               a.RightIssue.RightIssueLock,
			RightIssueNew:                a.RightIssue.RightIssueNew,
			RightIssueNewShare:           a.RightIssue.RightIssueNewShare,
			RightIssueOld:                a.RightIssue.RightIssueOld,
			RightIssuePrice:              a.RightIssue.RightIssuePrice,
			RightIssuePriceAdj:           a.RightIssue.RightIssuePriceAdj,
			RightIssuePriceFactor:        a.RightIssue.RightIssuePriceFactor,
			RightIssuePriceFormatted:     a.RightIssue.RightIssuePriceFormatted,
			RightIssueRatio:              a.RightIssue.RightIssueRatio,
			RightIssueRecdate:            a.RightIssue.RightIssueRecdate,
			RightIssueSubdate:            a.RightIssue.RightIssueSubdate,
			RightIssueTradingEnd:         a.RightIssue.RightIssueTradingEnd,
			RightIssueTradingStart:       a.RightIssue.RightIssueTradingStart,
			RightIssueForeignPercentage:  a.RightIssue.RightIssueForeignPercentage,
			RightIssueLocalPercentage:    a.RightIssue.RightIssueLocalPercentage,
			RightIssueNumberOfSecurities: a.RightIssue.RightIssueNumberOfSecurities,
			RightIssueTotal:              a.RightIssue.RightIssueTotal,
			EventNote:                    a.RightIssue.EventNote,
		}
		info.RightIssue = &out
	case a.StockSplit != nil:
		info.StockSplit = &stockSplitResponse{
			CompanyID:            a.StockSplit.CompanyID,
			CompanySymbol:        a.StockSplit.CompanySymbol,
			CorpActionActive:     a.StockSplit.CorpActionActive,
			StockSplitCreated:    a.StockSplit.StockSplitCreated,
			StockSplitCumdate:    a.StockSplit.StockSplitCumdate,
			StockSplitExdate:     a.StockSplit.StockSplitExdate,
			StockSplitFactor:     a.StockSplit.StockSplitFactor,
			StockSplitID:         a.StockSplit.StockSplitID,
			StockSplitIqpID:      a.StockSplit.StockSplitIqpID,
			StockSplitLastupdate: a.StockSplit.StockSplitLastupdate,
			StockSplitLock:       a.StockSplit.StockSplitLock,
			StockSplitNew:        a.StockSplit.StockSplitNew,
			StockSplitNewPrice:   a.StockSplit.StockSplitNewPrice,
			StockSplitNewShare:   a.StockSplit.StockSplitNewShare,
			StockSplitOld:        a.StockSplit.StockSplitOld,
			StockSplitRatio:      a.StockSplit.StockSplitRatio,
			StockSplitRecdate:    a.StockSplit.StockSplitRecdate,
			EventNote:            a.StockSplit.EventNote,
		}
	}
	b, _ := json.Marshal(info)
	return b
}
