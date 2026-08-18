package indexsummary

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

type IndexSummaryHandler struct {
	uc usecase.IndexSummaryUsecase
	v  validator.Validator
}

func NewIndexSummaryHandler(uc usecase.IndexSummaryUsecase, v validator.Validator) *IndexSummaryHandler {
	return &IndexSummaryHandler{uc: uc, v: v}
}

type indexSummaryRequest struct {
	Symbol   string `json:"symbol" validate:"required"`
	From     string `json:"from" validate:"omitempty,datetime=2006-01-02"`
	To       string `json:"to" validate:"omitempty,datetime=2006-01-02"`
	Interval string `json:"interval" validate:"omitempty"`
}

type indexSummaryResponse struct {
	Cagr                   string                      `json:"cagr"`
	Change                 float64                     `json:"change"`
	Drawdown               string                      `json:"drawdown"`
	MarkingPoint           string                      `json:"markingpoint"`
	Percentage             string                      `json:"percentage"`
	Timeframe              string                      `json:"timeframe"`
	XAxisOpt               string                      `json:"xaxisopt"`
	Previous               float64                     `json:"previous"`
	LineWeight             float64                     `json:"line_weight"`
	PreviousTimeframePrice indexSummaryPriceResponse   `json:"previous_timeframe_price"`
	ChartType              string                      `json:"chart_type"`
	IntervalInMinutes      int                         `json:"interval_in_minutes"`
	AllowedChartType       []string                    `json:"allowed_chart_type"`
	MaxCandles             int                         `json:"max_candles"`
	Prices                 []indexSummaryPriceResponse `json:"prices"`
}

type indexSummaryPriceResponse struct {
	Date          string  `json:"date"`
	FormattedDate string  `json:"formatted_date"`
	XLabel        string  `json:"xlabel"`
	Value         string  `json:"value"`
	Percentage    string  `json:"percentage"`
	Change        float64 `json:"change"`
	Open          string  `json:"open"`
	High          string  `json:"high"`
	Low           string  `json:"low"`
	Volume        string  `json:"volume"`
}

// indexChartResponse combines the index summary with chartbit OHLC bars.
type indexChartResponse struct {
	Summary indexSummaryResponse   `json:"summary"`
	Chart   indexChartOHLCResponse `json:"chart"`
}

type indexChartOHLCResponse struct {
	AllowDecimal int                      `json:"allow_decimal"`
	Chartbit     []indexChartItemResponse `json:"chartbit"`
}

type indexChartItemResponse struct {
	Date             string  `json:"date"`
	Unixdate         int64   `json:"unixdate"`
	Datetime         string  `json:"datetime"`
	UnixTimestamp    string  `json:"unix_timestamp"`
	Open             float64 `json:"open"`
	High             float64 `json:"high"`
	Low              float64 `json:"low"`
	Close            float64 `json:"close"`
	Volume           float64 `json:"volume"`
	Value            float64 `json:"value"`
	Frequency        float64 `json:"frequency"`
	ForeignBuy       float64 `json:"foreignbuy"`
	ForeignSell      float64 `json:"foreignsell"`
	ForeignFlow      float64 `json:"foreignflow"`
	SoxClose         float64 `json:"soxclose"`
	Dividend         float64 `json:"dividend"`
	ShareOutstanding float64 `json:"shareoutstanding"`
	FreqAnalyzer     float64 `json:"freq_analyzer"`
	Lot              float64 `json:"lot"`
	ForeignBuyToday  float64 `json:"foreign_buy"`
	ForeignSellToday float64 `json:"foreign_sell"`
	Symbol           string  `json:"symbol"`
}

// parseIndexSummaryRequest builds the request from the URL params shared by the
// summary and combined chart endpoints.
func parseIndexSummaryRequest(r *http.Request) indexSummaryRequest {
	req := indexSummaryRequest{Symbol: chi.URLParam(r, "symbol")}
	req.From = r.URL.Query().Get("from")
	req.To = r.URL.Query().Get("to")
	req.Interval = r.URL.Query().Get("interval")
	return req
}

// indexRangeRequirements returns per-field validation messages for range
// consistency, or nil when the range is well-formed. from/to must either both be
// provided or both omitted (omitted means the server defaults to the last
// trading session with data). When provided, from must not be later than to:
// the summary upstream 400s on reversed ranges ("Tanggal awal tidak boleh
// melebihi tanggal akhir"), which would otherwise surface as a confusing 500.
// Dates are validated as YYYY-MM-DD, so string comparison equals date ordering.
func indexRangeRequirements(req indexSummaryRequest) map[string]string {
	// Flag the field that is actually missing so the message points at the gap.
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

func (h *IndexSummaryHandler) IndexSummary(w http.ResponseWriter, r *http.Request) {
	req := parseIndexSummaryRequest(r)

	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate index summary params")
		return
	}
	if fields := indexRangeRequirements(req); len(fields) > 0 {
		response.ValidationError(w, "validation failed", fields)
		return
	}

	data, err := h.uc.GetIndexSummary(r.Context(), req.Symbol, req.From, req.To, req.Interval)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get index summary")
		return
	}
	response.OK(w, toResponse(data))
}

// IndexChart returns the index summary plus chartbit OHLC bars in one response.
// from/to are chronological (from = earlier, to = later); the chart range is
// swapped internally for chartbit's backward paging. interval only affects the
// summary part; the chart section is always daily OHLC bars.
func (h *IndexSummaryHandler) IndexChart(w http.ResponseWriter, r *http.Request) {
	req := parseIndexSummaryRequest(r)

	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate index chart params")
		return
	}
	if fields := indexRangeRequirements(req); len(fields) > 0 {
		response.ValidationError(w, "validation failed", fields)
		return
	}

	data, err := h.uc.GetIndexChart(r.Context(), req.Symbol, req.From, req.To, req.Interval)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get index chart")
		return
	}
	response.OK(w, toChartResponse(data))
}

func toChartResponse(d *domain.IndexChartData) indexChartResponse {
	out := indexChartResponse{
		Summary: toResponse(&d.Summary),
		Chart: indexChartOHLCResponse{
			AllowDecimal: d.Chart.AllowDecimal,
			Chartbit:     make([]indexChartItemResponse, 0, len(d.Chart.Chartbit)),
		},
	}
	for _, p := range d.Chart.Chartbit {
		out.Chart.Chartbit = append(out.Chart.Chartbit, indexChartItemResponse{
			Date:             p.Date,
			Unixdate:         p.Unixdate,
			Datetime:         p.Datetime,
			UnixTimestamp:    p.UnixTimestamp,
			Open:             p.Open,
			High:             p.High,
			Low:              p.Low,
			Close:            p.Close,
			Volume:           p.Volume,
			Value:            p.Value,
			Frequency:        p.Frequency,
			ForeignBuy:       p.ForeignBuy,
			ForeignSell:      p.ForeignSell,
			ForeignFlow:      p.ForeignFlow,
			SoxClose:         p.SoxClose,
			Dividend:         p.Dividend,
			ShareOutstanding: p.ShareOutstanding,
			FreqAnalyzer:     p.FreqAnalyzer,
			Lot:              p.Lot,
			ForeignBuyToday:  p.ForeignBuyToday,
			ForeignSellToday: p.ForeignSellToday,
			Symbol:           p.Symbol,
		})
	}
	return out
}

func toResponse(d *domain.IndexSummaryData) indexSummaryResponse {
	out := indexSummaryResponse{
		Cagr:                   d.Cagr,
		Change:                 d.Change,
		Drawdown:               d.Drawdown,
		MarkingPoint:           d.MarkingPoint,
		Percentage:             d.Percentage,
		Timeframe:              d.Timeframe,
		XAxisOpt:               d.XAxisOpt,
		Previous:               d.Previous,
		LineWeight:             d.LineWeight,
		PreviousTimeframePrice: toPriceResponse(d.PreviousTimeframePrice),
		ChartType:              d.ChartType,
		IntervalInMinutes:      d.IntervalInMinutes,
		AllowedChartType:       d.AllowedChartType,
		MaxCandles:             d.MaxCandles,
		Prices:                 make([]indexSummaryPriceResponse, 0, len(d.Prices)),
	}
	for _, p := range d.Prices {
		out.Prices = append(out.Prices, toPriceResponse(p))
	}
	return out
}

func toPriceResponse(p domain.IndexSummaryPrice) indexSummaryPriceResponse {
	return indexSummaryPriceResponse{
		Date:          p.Date,
		FormattedDate: p.FormattedDate,
		XLabel:        p.XLabel,
		Value:         p.Value,
		Percentage:    p.Percentage,
		Change:        p.Change,
		Open:          p.Open,
		High:          p.High,
		Low:           p.Low,
		Volume:        p.Volume,
	}
}
