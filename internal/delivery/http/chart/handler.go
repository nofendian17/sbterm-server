package chart

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/response"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

type ChartbitHandler struct {
	uc usecase.ChartbitUsecase
	v  validator.Validator
}

func NewChartbitHandler(uc usecase.ChartbitUsecase, v validator.Validator) *ChartbitHandler {
	return &ChartbitHandler{uc: uc, v: v}
}

type chartPriceRequest struct {
	Symbol    string `json:"symbol" validate:"required"`
	Timeframe string `json:"timeframe" validate:"required,oneof=daily intraday"`
	From      string `json:"from" validate:"omitempty"`
	To        string `json:"to" validate:"omitempty"`
	Limit     int    `json:"limit" validate:"omitempty"`
}

type chartPriceResponse struct {
	AllowDecimal int                 `json:"allow_decimal"`
	Chartbit     []chartItemResponse `json:"chartbit"`
}

type chartItemResponse struct {
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

func (h *ChartbitHandler) ChartPrice(w http.ResponseWriter, r *http.Request) {
	req := chartPriceRequest{Symbol: chi.URLParam(r, "symbol"), Timeframe: r.URL.Query().Get("timeframe")}
	req.From = r.URL.Query().Get("from")
	req.To = r.URL.Query().Get("to")
	if v := r.URL.Query().Get("limit"); v != "" {
		req.Limit, _ = strconv.Atoi(v)
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate chart params")
		return
	}

	data, err := h.uc.GetChartPrice(r.Context(), req.Symbol, req.Timeframe, req.From, req.To, req.Limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get chart price")
		return
	}
	response.OK(w, toResponse(data))
}

func toResponse(d *domain.ChartPriceData) chartPriceResponse {
	out := chartPriceResponse{AllowDecimal: d.AllowDecimal, Chartbit: make([]chartItemResponse, 0, len(d.Chartbit))}
	for _, p := range d.Chartbit {
		out.Chartbit = append(out.Chartbit, chartItemResponse{
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
