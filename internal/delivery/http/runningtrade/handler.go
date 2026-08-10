package runningtrade

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/response"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

// Default values applied when the corresponding query param is omitted.
const (
	defaultInvestorType = "INVESTOR_TYPE_ALL"
	defaultMarketBoard  = "BOARD_TYPE_ALL"
	defaultPeriod       = "RT_PERIOD_LAST_1_DAY"
)

type RunningTradeHandler struct {
	uc usecase.RunningTradeUsecase
	v  validator.Validator
}

func NewRunningTradeHandler(uc usecase.RunningTradeUsecase, v validator.Validator) *RunningTradeHandler {
	return &RunningTradeHandler{uc: uc, v: v}
}

type runningTradeRequest struct {
	Symbol       string   `json:"symbol" validate:"required"`
	BrokerCodes  []string `json:"broker_code"`
	From         string   `json:"from" validate:"omitempty,datetime=2006-01-02"`
	To           string   `json:"to" validate:"omitempty,datetime=2006-01-02"`
	InvestorType string   `json:"investor_type" validate:"omitempty,oneof=INVESTOR_TYPE_ALL INVESTOR_TYPE_FOREIGN INVESTOR_TYPE_DOMESTIC"`
	MarketBoard  string   `json:"market_board" validate:"omitempty,oneof=BOARD_TYPE_ALL BOARD_TYPE_REGULAR BOARD_TYPE_CASH BOARD_TYPE_NEGOTIATION"`
	Period       string   `json:"period" validate:"omitempty,oneof=RT_PERIOD_LAST_1_DAY RT_PERIOD_LAST_7_DAYS RT_PERIOD_LAST_1_MONTH RT_PERIOD_LAST_3_MONTHS RT_PERIOD_YEAR_TO_DATE RT_PERIOD_LAST_1_YEAR"`
}

type runningTradeResponse struct {
	From            string                        `json:"from"`
	To              string                        `json:"to"`
	DataLastUpdated string                        `json:"data_last_updated"`
	PriceChartData  []runningTradePricePointResp  `json:"price_chart_data"`
	BrokerChartData []runningTradeBrokerGroupResp `json:"broker_chart_data"`
	DateSessionInfo string                        `json:"date_session_info"`
}

type runningTradePricePointResp struct {
	Date          string        `json:"date"`
	Time          string        `json:"time"`
	Value         rawFormatted  `json:"value"`
	DatetimeLabel string        `json:"datetime_label"`
	Open          *rawFormatted `json:"open"`
	High          *rawFormatted `json:"high"`
	Low           *rawFormatted `json:"low"`
}

type runningTradeBrokerGroupResp struct {
	Type    string                        `json:"type"`
	Brokers []string                      `json:"brokers"`
	Charts  []runningTradeBrokerChartResp `json:"charts"`
}

type runningTradeBrokerChartResp struct {
	BrokerCode string                       `json:"broker_code"`
	Chart      []runningTradeChartPointResp `json:"chart"`
}

type runningTradeChartPointResp struct {
	Date          string        `json:"date"`
	Time          string        `json:"time"`
	Value         rawFormatted  `json:"value"`
	DatetimeLabel string        `json:"datetime_label"`
	Open          *rawFormatted `json:"open"`
	High          *rawFormatted `json:"high"`
	Low           *rawFormatted `json:"low"`
}

type rawFormatted struct {
	Raw       string `json:"raw"`
	Formatted string `json:"formatted"`
}

// runningTradeRangeRequirements returns per-field validation messages for the
// range/period combination, or nil when well-formed. from/to must either both be
// provided or both omitted (when both are omitted the period enum selects the
// timeframe, defaulting to the last 1 day). A reversed range (from later than
// to) is rejected because the upstream 400s on it, which would otherwise
// surface as a confusing 500. Dates are validated as YYYY-MM-DD, so string
// comparison equals date ordering.
func runningTradeRangeRequirements(req runningTradeRequest) map[string]string {
	if (req.From == "") != (req.To == "") {
		if req.From == "" {
			return map[string]string{"from": "from and to must both be provided or both omitted"}
		}
		return map[string]string{"to": "from and to must both be provided or both omitted"}
	}
	if req.From != "" && req.From > req.To {
		return map[string]string{"from": "must be earlier than or equal to to"}
	}
	return nil
}

func (h *RunningTradeHandler) RunningTrade(w http.ResponseWriter, r *http.Request) {
	req := runningTradeRequest{
		Symbol:       chi.URLParam(r, "symbol"),
		BrokerCodes:  r.URL.Query()["broker_code"],
		From:         r.URL.Query().Get("from"),
		To:           r.URL.Query().Get("to"),
		InvestorType: r.URL.Query().Get("investor_type"),
		MarketBoard:  r.URL.Query().Get("market_board"),
		Period:       r.URL.Query().Get("period"),
	}
	if req.InvestorType == "" {
		req.InvestorType = defaultInvestorType
	}
	if req.MarketBoard == "" {
		req.MarketBoard = defaultMarketBoard
	}
	if req.Period == "" && req.From == "" && req.To == "" {
		req.Period = defaultPeriod
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate running trade params")
		return
	}
	if fields := runningTradeRangeRequirements(req); len(fields) > 0 {
		response.ValidationError(w, "validation failed", fields)
		return
	}

	data, err := h.uc.GetRunningTradeChart(r.Context(), req.Symbol, req.BrokerCodes, req.From, req.To, req.InvestorType, req.MarketBoard, req.Period)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get running trade chart")
		return
	}
	response.OK(w, toResponse(data))
}

func toResponse(d *domain.RunningTradeData) runningTradeResponse {
	out := runningTradeResponse{
		From:            d.From,
		To:              d.To,
		DataLastUpdated: d.DataLastUpdated,
		PriceChartData:  make([]runningTradePricePointResp, 0, len(d.PriceChartData)),
		BrokerChartData: make([]runningTradeBrokerGroupResp, 0, len(d.BrokerChartData)),
		DateSessionInfo: d.DateSessionInfo,
	}
	for _, p := range d.PriceChartData {
		out.PriceChartData = append(out.PriceChartData, runningTradePricePointResp{
			Date:          p.Date,
			Time:          p.Time,
			Value:         toRawFormatted(p.Value),
			DatetimeLabel: p.DatetimeLabel,
			Open:          toRawFormattedPtr(p.Open),
			High:          toRawFormattedPtr(p.High),
			Low:           toRawFormattedPtr(p.Low),
		})
	}
	for _, g := range d.BrokerChartData {
		group := runningTradeBrokerGroupResp{
			Type:    g.Type,
			Brokers: g.Brokers,
			Charts:  make([]runningTradeBrokerChartResp, 0, len(g.Charts)),
		}
		for _, ch := range g.Charts {
			group.Charts = append(group.Charts, runningTradeBrokerChartResp{
				BrokerCode: ch.BrokerCode,
				Chart:      toChartPointResponses(ch.Chart),
			})
		}
		out.BrokerChartData = append(out.BrokerChartData, group)
	}
	return out
}

func toChartPointResponses(points []domain.RunningTradeChartPoint) []runningTradeChartPointResp {
	out := make([]runningTradeChartPointResp, 0, len(points))
	for _, p := range points {
		out = append(out, runningTradeChartPointResp{
			Date:          p.Date,
			Time:          p.Time,
			Value:         toRawFormatted(p.Value),
			DatetimeLabel: p.DatetimeLabel,
			Open:          toRawFormattedPtr(p.Open),
			High:          toRawFormattedPtr(p.High),
			Low:           toRawFormattedPtr(p.Low),
		})
	}
	return out
}

func toRawFormatted(v domain.RawFormatted) rawFormatted {
	return rawFormatted{Raw: v.Raw, Formatted: v.Formatted}
}

func toRawFormattedPtr(v *domain.RawFormatted) *rawFormatted {
	if v == nil {
		return nil
	}
	rf := toRawFormatted(*v)
	return &rf
}
