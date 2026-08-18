package activity

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

// Default values applied when the corresponding query param is omitted.
const (
	defaultInvestorType          = "INVESTOR_TYPE_ALL"
	defaultChartMarketBoard      = "BOARD_TYPE_ALL"
	defaultMarketBoard           = "MARKET_TYPE_REGULER"
	defaultTransactionType       = "TRANSACTION_TYPE_GROSS"
	defaultLimit                 = 20
	defaultPage                  = 1
	defaultNetValPeriod          = "NET_VAL_PERIOD_7D"
	defaultPeriod                = "RT_PERIOD_LAST_1_DAY"
	defaultInterval              = "INTERVAL_DAILY"
	defaultNetInterval           = "INTERVAL_MONTHLY"
	defaultHistoricalMarketBoard = "BOARD_TYPE_ALL"
)

type ActivityHandler struct {
	uc usecase.ActivityUsecase
	v  validator.Validator
}

func NewActivityHandler(uc usecase.ActivityUsecase, v validator.Validator) *ActivityHandler {
	return &ActivityHandler{uc: uc, v: v}
}

type activityChartRequest struct {
	Symbols      []string `json:"symbols"`
	BrokersCode  []string `json:"brokers_code"`
	From         string   `json:"from" validate:"omitempty,datetime=2006-01-02"`
	To           string   `json:"to" validate:"omitempty,datetime=2006-01-02"`
	Period       string   `json:"period" validate:"omitempty,oneof=RT_PERIOD_LAST_1_DAY RT_PERIOD_LAST_7_DAYS RT_PERIOD_LAST_1_MONTH RT_PERIOD_LAST_3_MONTHS RT_PERIOD_YEAR_TO_DATE RT_PERIOD_LAST_1_YEAR"`
	InvestorType string   `json:"investor_type" validate:"omitempty,oneof=INVESTOR_TYPE_ALL INVESTOR_TYPE_FOREIGN INVESTOR_TYPE_DOMESTIC"`
	MarketBoard  string   `json:"market_board" validate:"omitempty,oneof=BOARD_TYPE_ALL BOARD_TYPE_REGULAR BOARD_TYPE_CASH BOARD_TYPE_NEGOTIATION"`
}

type activityRequest struct {
	BrokerCode      []string `json:"broker_code"`
	TransactionType string   `json:"transaction_type" validate:"omitempty,oneof=TRANSACTION_TYPE_GROSS TRANSACTION_TYPE_NET"`
	From            string   `json:"from" validate:"omitempty,datetime=2006-01-02"`
	To              string   `json:"to" validate:"omitempty,datetime=2006-01-02"`
	InvestorType    string   `json:"investor_type" validate:"omitempty,oneof=INVESTOR_TYPE_ALL INVESTOR_TYPE_FOREIGN INVESTOR_TYPE_DOMESTIC"`
	MarketBoard     string   `json:"market_board" validate:"omitempty,oneof=MARKET_TYPE_ALL MARKET_TYPE_REGULER MARKET_TYPE_NEGO"`
	NetValPeriod    string   `json:"net_val_period" validate:"omitempty,oneof=NET_VAL_PERIOD_7D NET_VAL_PERIOD_1M NET_VAL_PERIOD_3M"`
	Limit           int      `json:"limit" validate:"min=1"`
	Page            int      `json:"page" validate:"min=1"`
}

type activityHistoricalRequest struct {
	Interval     string   `json:"interval" validate:"omitempty,oneof=INTERVAL_DAILY INTERVAL_WEEKLY INTERVAL_MONTHLY"`
	DateFrom     string   `json:"date_from" validate:"omitempty,datetime=2006-01-02"`
	DateTo       string   `json:"date_to" validate:"omitempty,datetime=2006-01-02"`
	BrokerCodes  []string `json:"broker_codes"`
	Symbols      []string `json:"symbols"`
	MarketBoard  string   `json:"market_board" validate:"omitempty,oneof=BOARD_TYPE_ALL BOARD_TYPE_REGULAR BOARD_TYPE_CASH BOARD_TYPE_NEGOTIATION"`
	InvestorType string   `json:"investor_type" validate:"omitempty,oneof=INVESTOR_TYPE_ALL INVESTOR_TYPE_FOREIGN INVESTOR_TYPE_DOMESTIC"`
	NetInterval  string   `json:"net_interval" validate:"omitempty,oneof=INTERVAL_DAILY INTERVAL_WEEKLY INTERVAL_MONTHLY"`
}

type activityChartResponse struct {
	From            string                   `json:"from"`
	To              string                   `json:"to"`
	DataLastUpdated string                   `json:"data_last_updated"`
	ChartData       []activityChartGroupResp `json:"chart_data"`
	DateSessionInfo string                   `json:"date_session_info"`
	BrokerCode      []string                 `json:"broker_code"`
	BrokerName      string                   `json:"broker_name"`
}

type activityChartGroupResp struct {
	Type    string                    `json:"type"`
	Symbols []string                  `json:"symbols"`
	Charts  []activityChartSeriesResp `json:"charts"`
}

type activityChartSeriesResp struct {
	Symbol string                   `json:"symbol"`
	Chart  []activityChartPointResp `json:"chart"`
}

type activityChartPointResp struct {
	Date          string       `json:"date"`
	Time          string       `json:"time"`
	Value         rawFormatted `json:"value"`
	DatetimeLabel string       `json:"datetime_label"`
}

type activityResponse struct {
	BrokerActivityTransaction activityBrokerTransactionResp `json:"broker_activity_transaction"`
	From                      string                        `json:"from"`
	To                        string                        `json:"to"`
	BrokerCode                string                        `json:"broker_code"`
	BrokerName                string                        `json:"broker_name"`
}

type activityBrokerTransactionResp struct {
	BrokersBuy  []activityBrokerResp `json:"brokers_buy"`
	BrokersSell []activityBrokerResp `json:"brokers_sell"`
}

type activityBrokerResp struct {
	StockCode     string                      `json:"stock_code"`
	BrokerCode    string                      `json:"broker_code"`
	Type          string                      `json:"type"`
	Date          string                      `json:"date"`
	Value         float64                     `json:"value"`
	Lot           float64                     `json:"lot"`
	AveragePrice  float64                     `json:"avg_price"`
	Frequency     float64                     `json:"freq"`
	CompanyDetail activityCompanyDetailResp   `json:"company_detail"`
	NetValueTrend []activityNetValueTrendResp `json:"nval_trend"`
}

type activityCompanyDetailResp struct {
	IconURL    string                 `json:"icon_url"`
	CorpAction activityCorpActionResp `json:"corpaction"`
	Notation   []activityNotationResp `json:"notation"`
}

type activityCorpActionResp struct {
	Active bool   `json:"active"`
	Icon   string `json:"icon"`
	Text   string `json:"text"`
}

type activityNotationResp struct {
	NotationCode string                   `json:"notation_code"`
	NotationDesc string                   `json:"notation_desc"`
	IconURL      activityNotationIconResp `json:"icon_url"`
}

type activityNotationIconResp struct {
	LightMode string `json:"light_mode"`
	DarkMode  string `json:"dark_mode"`
}

type activityNetValueTrendResp struct {
	Date  string  `json:"date"`
	NVal  float64 `json:"nval"`
	NVol  float64 `json:"nvol"`
	NFreq float64 `json:"nfreq"`
}

type activityHistoricalResponse struct {
	DateFrom    string                         `json:"date_from"`
	DateTo      string                         `json:"date_to"`
	Symbols     []string                       `json:"symbols"`
	BrokerCodes []string                       `json:"broker_codes"`
	BrokerName  string                         `json:"broker_name"`
	Records     []activityHistoricalRecordResp `json:"records"`
	Pagination  activityHistoricalPaginateResp `json:"pagination"`
	Summary     activityHistoricalSummaryResp  `json:"summary"`
}

type activityHistoricalRecordResp struct {
	Date          string                      `json:"date"`
	BrokerCode    string                      `json:"broker_code"`
	TradeActivity activityHistoricalTradeResp `json:"trade_activity"`
	PriceActivity activityHistoricalPriceResp `json:"price_activity"`
}

type activityHistoricalTradeResp struct {
	NetSummary     activitySummaryResp        `json:"net_summary"`
	BuySummary     activitySummaryResp        `json:"buy_summary"`
	SellSummary    activitySummaryResp        `json:"sell_summary"`
	ForeignSummary activityForeignSummaryResp `json:"foreign_summary"`
	TotalBuyLot    activityLotShareResp       `json:"total_buy_lot"`
	TotalSellLot   activityLotShareResp       `json:"total_sell_lot"`
}

type activitySummaryResp struct {
	AveragePrice float64 `json:"avg_price"`
	Frequency    float64 `json:"freq"`
	Lot          float64 `json:"lot"`
	Value        float64 `json:"value"`
}

type activityForeignSummaryResp struct {
	ForeignBuy  float64 `json:"foreign_buy"`
	ForeignSell float64 `json:"foreign_sell"`
	NetForeign  float64 `json:"net_foreign"`
}

type activityLotShareResp struct {
	Amount float64 `json:"amount"`
	Pct    float64 `json:"pct"`
}

type activityHistoricalPriceResp struct {
	ClosePrice    string                        `json:"close_price"`
	ReturnSummary activityHistoricalPriceReturn `json:"return_summary"`
}

type activityHistoricalPriceReturn struct {
	Amount float64 `json:"amount"`
	Pct    float64 `json:"pct"`
}

type activityHistoricalPaginateResp struct {
	Page    int  `json:"page"`
	Limit   int  `json:"limit"`
	HasNext bool `json:"has_next"`
	HasPrev bool `json:"has_prev"`
}

type activityHistoricalSummaryResp struct {
	GroupType string                               `json:"group_type"`
	Data      []activityHistoricalSummaryGroupResp `json:"data"`
}

type activityHistoricalSummaryGroupResp struct {
	DateFrom   string              `json:"date_from"`
	DateTo     string              `json:"date_to"`
	NetSummary activitySummaryResp `json:"net_summary"`
}

type rawFormatted struct {
	Raw       string `json:"raw"`
	Formatted string `json:"formatted"`
}

func parseIntQuery(s string, def int) (int, bool) {
	if s == "" {
		return def, true
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

func (h *ActivityHandler) ActivityChart(w http.ResponseWriter, r *http.Request) {
	req := activityChartRequest{
		Symbols:      r.URL.Query()["symbols"],
		BrokersCode:  r.URL.Query()["brokers_code"],
		From:         r.URL.Query().Get("from"),
		To:           r.URL.Query().Get("to"),
		Period:       r.URL.Query().Get("period"),
		InvestorType: r.URL.Query().Get("investor_type"),
		MarketBoard:  r.URL.Query().Get("market_board"),
	}
	if req.InvestorType == "" {
		req.InvestorType = defaultInvestorType
	}
	if req.MarketBoard == "" {
		req.MarketBoard = defaultChartMarketBoard
	}
	if req.Period == "" && req.From == "" && req.To == "" {
		req.Period = defaultPeriod
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate activity chart params")
		return
	}

	data, err := h.uc.GetActivityChart(r.Context(), req.Symbols, req.BrokersCode, req.From, req.To, req.Period, req.InvestorType, req.MarketBoard)
	if err != nil {
		var upErr *domain.UpstreamError
		if errors.As(err, &upErr) && upErr.Status == http.StatusBadRequest {
			response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "no activity chart data for the requested date range")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get activity chart")
		return
	}
	response.OK(w, toChartResponse(data))
}

func (h *ActivityHandler) Activity(w http.ResponseWriter, r *http.Request) {
	limit, ok := parseIntQuery(r.URL.Query().Get("limit"), defaultLimit)
	if !ok {
		response.ValidationError(w, "validation failed", map[string]string{"limit": "must be a valid integer"})
		return
	}
	page, ok := parseIntQuery(r.URL.Query().Get("page"), defaultPage)
	if !ok {
		response.ValidationError(w, "validation failed", map[string]string{"page": "must be a valid integer"})
		return
	}
	req := activityRequest{
		BrokerCode:      r.URL.Query()["broker_code"],
		TransactionType: r.URL.Query().Get("transaction_type"),
		From:            r.URL.Query().Get("from"),
		To:              r.URL.Query().Get("to"),
		InvestorType:    r.URL.Query().Get("investor_type"),
		MarketBoard:     r.URL.Query().Get("market_board"),
		NetValPeriod:    r.URL.Query().Get("net_val_period"),
		Limit:           limit,
		Page:            page,
	}
	if req.InvestorType == "" {
		req.InvestorType = defaultInvestorType
	}
	if req.MarketBoard == "" {
		req.MarketBoard = defaultMarketBoard
	}
	if req.TransactionType == "" {
		req.TransactionType = defaultTransactionType
	}
	if req.NetValPeriod == "" {
		req.NetValPeriod = defaultNetValPeriod
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate activity params")
		return
	}

	data, err := h.uc.GetActivity(r.Context(), req.BrokerCode, req.TransactionType, req.InvestorType, req.MarketBoard, req.Limit, req.Page, req.From, req.To, req.NetValPeriod)
	if err != nil {
		var upErr *domain.UpstreamError
		if errors.As(err, &upErr) && upErr.Status == http.StatusBadRequest {
			response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "no activity data for the requested date range")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get activity")
		return
	}
	response.OK(w, toActivityResponse(data))
}

func (h *ActivityHandler) ActivityHistorical(w http.ResponseWriter, r *http.Request) {
	req := activityHistoricalRequest{
		Interval:     r.URL.Query().Get("interval"),
		DateFrom:     r.URL.Query().Get("date_from"),
		DateTo:       r.URL.Query().Get("date_to"),
		BrokerCodes:  r.URL.Query()["broker_codes"],
		Symbols:      r.URL.Query()["symbols"],
		MarketBoard:  r.URL.Query().Get("market_board"),
		InvestorType: r.URL.Query().Get("investor_type"),
		NetInterval:  r.URL.Query().Get("net_interval"),
	}
	if req.Interval == "" {
		req.Interval = defaultInterval
	}
	if req.NetInterval == "" {
		req.NetInterval = defaultNetInterval
	}
	if req.InvestorType == "" {
		req.InvestorType = defaultInvestorType
	}
	if req.MarketBoard == "" {
		req.MarketBoard = defaultHistoricalMarketBoard
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate activity historical params")
		return
	}

	data, err := h.uc.GetActivityHistorical(r.Context(), req.Interval, req.DateFrom, req.DateTo, req.BrokerCodes, req.Symbols, req.MarketBoard, req.InvestorType, req.NetInterval)
	if err != nil {
		var upErr *domain.UpstreamError
		if errors.As(err, &upErr) && upErr.Status == http.StatusBadRequest {
			response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "no broker activity historical data for the requested parameters")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get broker activity historical")
		return
	}
	response.OK(w, toHistoricalResponse(data))
}

func toChartResponse(d *domain.ActivityChartData) activityChartResponse {
	out := activityChartResponse{
		From:            d.From,
		To:              d.To,
		DataLastUpdated: d.DataLastUpdated,
		ChartData:       make([]activityChartGroupResp, 0, len(d.ChartData)),
		DateSessionInfo: d.DateSessionInfo,
		BrokerCode:      d.BrokerCode,
		BrokerName:      d.BrokerName,
	}
	for _, g := range d.ChartData {
		group := activityChartGroupResp{
			Type:    g.Type,
			Symbols: g.Symbols,
			Charts:  make([]activityChartSeriesResp, 0, len(g.Charts)),
		}
		for _, ch := range g.Charts {
			series := activityChartSeriesResp{
				Symbol: ch.Symbol,
				Chart:  make([]activityChartPointResp, 0, len(ch.Chart)),
			}
			for _, p := range ch.Chart {
				series.Chart = append(series.Chart, activityChartPointResp{
					Date:          p.Date,
					Time:          p.Time,
					Value:         rawFormatted{Raw: p.Value.Raw, Formatted: p.Value.Formatted},
					DatetimeLabel: p.DatetimeLabel,
				})
			}
			group.Charts = append(group.Charts, series)
		}
		out.ChartData = append(out.ChartData, group)
	}
	return out
}

func toActivityResponse(d *domain.ActivityData) activityResponse {
	ba := d.BrokerActivityTransaction
	return activityResponse{
		BrokerActivityTransaction: activityBrokerTransactionResp{
			BrokersBuy:  toBrokerResponses(ba.BrokersBuy),
			BrokersSell: toBrokerResponses(ba.BrokersSell),
		},
		From:       d.From,
		To:         d.To,
		BrokerCode: d.BrokerCode,
		BrokerName: d.BrokerName,
	}
}

func toBrokerResponses(in []domain.BrokerActivity) []activityBrokerResp {
	out := make([]activityBrokerResp, 0, len(in))
	for _, b := range in {
		out = append(out, activityBrokerResp{
			StockCode:    b.StockCode,
			BrokerCode:   b.BrokerCode,
			Type:         b.Type,
			Date:         b.Date,
			Value:        b.Value,
			Lot:          b.Lot,
			AveragePrice: b.AveragePrice,
			Frequency:    b.Frequency,
			CompanyDetail: activityCompanyDetailResp{
				IconURL:    b.CompanyDetail.IconURL,
				CorpAction: activityCorpActionResp{Active: b.CompanyDetail.CorpAction.Active, Icon: b.CompanyDetail.CorpAction.Icon, Text: b.CompanyDetail.CorpAction.Text},
				Notation:   toNotationResponses(b.CompanyDetail.Notation),
			},
			NetValueTrend: toNetValueTrendResponses(b.NetValueTrend),
		})
	}
	return out
}

func toNotationResponses(in []domain.ActivityNotation) []activityNotationResp {
	out := make([]activityNotationResp, 0, len(in))
	for _, n := range in {
		out = append(out, activityNotationResp{
			NotationCode: n.NotationCode,
			NotationDesc: n.NotationDesc,
			IconURL:      activityNotationIconResp{LightMode: n.IconURL.LightMode, DarkMode: n.IconURL.DarkMode},
		})
	}
	return out
}

func toNetValueTrendResponses(in []domain.ActivityNetValueTrend) []activityNetValueTrendResp {
	out := make([]activityNetValueTrendResp, 0, len(in))
	for _, n := range in {
		out = append(out, activityNetValueTrendResp{
			Date:  n.Date,
			NVal:  n.NVal,
			NVol:  n.NVol,
			NFreq: n.NFreq,
		})
	}
	return out
}

func toHistoricalResponse(d *domain.ActivityHistoricalData) activityHistoricalResponse {
	out := activityHistoricalResponse{
		DateFrom:    d.DateFrom,
		DateTo:      d.DateTo,
		Symbols:     d.Symbols,
		BrokerCodes: d.BrokerCodes,
		BrokerName:  d.BrokerName,
		Records:     make([]activityHistoricalRecordResp, 0, len(d.Records)),
		Pagination: activityHistoricalPaginateResp{
			Page:    d.Pagination.Page,
			Limit:   d.Pagination.Limit,
			HasNext: d.Pagination.HasNext,
			HasPrev: d.Pagination.HasPrev,
		},
		Summary: activityHistoricalSummaryResp{
			GroupType: d.Summary.GroupType,
			Data:      make([]activityHistoricalSummaryGroupResp, 0, len(d.Summary.Data)),
		},
	}
	for _, rec := range d.Records {
		out.Records = append(out.Records, activityHistoricalRecordResp{
			Date:       rec.Date,
			BrokerCode: rec.BrokerCode,
			TradeActivity: activityHistoricalTradeResp{
				NetSummary:     toHistoricalSummaryResp(rec.TradeActivity.NetSummary),
				BuySummary:     toHistoricalSummaryResp(rec.TradeActivity.BuySummary),
				SellSummary:    toHistoricalSummaryResp(rec.TradeActivity.SellSummary),
				ForeignSummary: toHistoricalForeignSummaryResp(rec.TradeActivity.ForeignSummary),
				TotalBuyLot:    toHistoricalLotShareResp(rec.TradeActivity.TotalBuyLot),
				TotalSellLot:   toHistoricalLotShareResp(rec.TradeActivity.TotalSellLot),
			},
			PriceActivity: activityHistoricalPriceResp{
				ClosePrice: rec.PriceActivity.ClosePrice,
				ReturnSummary: activityHistoricalPriceReturn{
					Amount: rec.PriceActivity.ReturnSummary.Amount,
					Pct:    rec.PriceActivity.ReturnSummary.Pct,
				},
			},
		})
	}
	for _, g := range d.Summary.Data {
		out.Summary.Data = append(out.Summary.Data, activityHistoricalSummaryGroupResp{
			DateFrom:   g.DateFrom,
			DateTo:     g.DateTo,
			NetSummary: toHistoricalSummaryResp(g.NetSummary),
		})
	}
	return out
}

func toHistoricalSummaryResp(in domain.ActivitySummary) activitySummaryResp {
	return activitySummaryResp{
		AveragePrice: in.AveragePrice,
		Frequency:    in.Frequency,
		Lot:          in.Lot,
		Value:        in.Value,
	}
}

func toHistoricalForeignSummaryResp(in domain.ActivityForeignSummary) activityForeignSummaryResp {
	return activityForeignSummaryResp{
		ForeignBuy:  in.ForeignBuy,
		ForeignSell: in.ForeignSell,
		NetForeign:  in.NetForeign,
	}
}

func toHistoricalLotShareResp(in domain.ActivityLotShare) activityLotShareResp {
	return activityLotShareResp{
		Amount: in.Amount,
		Pct:    in.Pct,
	}
}
