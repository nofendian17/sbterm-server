package runningtrade

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

// Default values applied when the corresponding query param is omitted.
const (
	defaultInvestorType = "INVESTOR_TYPE_ALL"
	defaultMarketBoard  = "BOARD_TYPE_ALL"
	defaultPeriod       = "RT_PERIOD_LAST_1_DAY"
	defaultSort         = "ASC"
	defaultOrderBy      = "RUNNING_TRADE_ORDER_BY_TIME"
	defaultFeedLimit    = 80
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

type runningTradeFeedRequest struct {
	Symbol      string `json:"symbol" validate:"required"`
	Sort        string `json:"sort" validate:"omitempty,oneof=ASC DESC"`
	OrderBy     string `json:"order_by" validate:"omitempty,oneof=RUNNING_TRADE_ORDER_BY_TIME RUNNING_TRADE_ORDER_BY_LOT RUNNING_TRADE_ORDER_BY_VALUE"`
	Date        string `json:"date" validate:"omitempty,datetime=2006-01-02"`
	Limit       int    `validate:"min=1"`
	TradeNumber int64
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

type runningTradeFeedResponse struct {
	IsOpenMarket bool                           `json:"is_open_market"`
	RunningTrade []runningTradeFeedItemResponse `json:"running_trade"`
}

type runningTradeFeedItemResponse struct {
	ID               string                        `json:"id"`
	Time             string                        `json:"time"`
	Action           string                        `json:"action"`
	Code             string                        `json:"code"`
	Price            string                        `json:"price"`
	Change           string                        `json:"change"`
	Lot              string                        `json:"lot"`
	IsBrokerExists   bool                          `json:"is_broker_exists"`
	Buyer            string                        `json:"buyer"`
	Seller           string                        `json:"seller"`
	TradeNumber      string                        `json:"trade_number"`
	BuyerType        string                        `json:"buyer_type"`
	SellerType       string                        `json:"seller_type"`
	MarketBoard      string                        `json:"market_board"`
	BuyOrderNumber   string                        `json:"buy_order_number"`
	SellOrderNumber  string                        `json:"sell_order_number"`
	GroupOrderNumber string                        `json:"group_order_number"`
	Value            runningTradeFeedValueResponse `json:"value"`
}

type runningTradeFeedValueResponse struct {
	Raw       json.Number `json:"raw"`
	Formatted string      `json:"formatted"`
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

func (h *RunningTradeHandler) RunningTradeChart(w http.ResponseWriter, r *http.Request) {
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
		// The upstream 400s on ranges whose session has no data yet (e.g.
		// today before the market closes). Surface that as a clear 422 instead
		// of a confusing 500.
		var upErr *domain.UpstreamError
		if errors.As(err, &upErr) && upErr.Status == http.StatusBadRequest {
			response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "no running trade data for the requested date range")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get running trade chart")
		return
	}
	response.OK(w, toResponse(data))
}

func (h *RunningTradeHandler) RunningTrade(w http.ResponseWriter, r *http.Request) {
	req := runningTradeFeedRequest{
		Symbol:  r.URL.Query().Get("symbol"),
		Sort:    r.URL.Query().Get("sort"),
		OrderBy: r.URL.Query().Get("order_by"),
		Date:    r.URL.Query().Get("date"),
	}
	limit, err := parseIntQuery(r.URL.Query().Get("limit"), defaultFeedLimit)
	if err != nil {
		response.ValidationError(w, "validation failed", map[string]string{"limit": "must be a valid integer"})
		return
	}
	req.Limit = limit

	tradeNumber, err := parseInt64Query(r.URL.Query().Get("trade_number"))
	if err != nil {
		response.ValidationError(w, "validation failed", map[string]string{"trade_number": "must be a valid integer"})
		return
	}
	req.TradeNumber = tradeNumber

	if req.Sort == "" {
		req.Sort = defaultSort
	}
	if req.OrderBy == "" {
		req.OrderBy = defaultOrderBy
	}

	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate running trade params")
		return
	}

	// An empty date is omitted upstream so it falls back to the most recent
	// session with data.
	data, err := h.uc.GetRunningTrade(r.Context(), req.Symbol, req.Sort, req.OrderBy, req.Date, req.Limit, req.TradeNumber)
	if err != nil {
		var upErr *domain.UpstreamError
		if errors.As(err, &upErr) && upErr.Status == http.StatusBadRequest {
			response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "no running trade data for the requested parameters")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get running trade")
		return
	}
	response.OK(w, toRunningTradeFeedResponse(data))
}

func toRunningTradeFeedResponse(d *domain.RunningTradeFeed) runningTradeFeedResponse {
	out := runningTradeFeedResponse{
		IsOpenMarket: d.IsOpenMarket,
		RunningTrade: make([]runningTradeFeedItemResponse, 0, len(d.RunningTrade)),
	}
	for _, t := range d.RunningTrade {
		out.RunningTrade = append(out.RunningTrade, runningTradeFeedItemResponse{
			ID:               t.ID,
			Time:             t.Time,
			Action:           t.Action,
			Code:             t.Code,
			Price:            t.Price,
			Change:           t.Change,
			Lot:              t.Lot,
			IsBrokerExists:   t.IsBrokerExists,
			Buyer:            t.Buyer,
			Seller:           t.Seller,
			TradeNumber:      t.TradeNumber,
			BuyerType:        t.BuyerType,
			SellerType:       t.SellerType,
			MarketBoard:      t.MarketBoard,
			BuyOrderNumber:   t.BuyOrderNumber,
			SellOrderNumber:  t.SellOrderNumber,
			GroupOrderNumber: t.GroupOrderNumber,
			Value:            runningTradeFeedValueResponse{Raw: t.Value.Raw, Formatted: t.Value.Formatted},
		})
	}
	return out
}

func parseIntQuery(s string, def int) (int, error) {
	if s == "" {
		return def, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func parseInt64Query(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
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
