package priceperformance

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

type PricePerformanceHandler struct {
	uc usecase.PricePerformanceUsecase
	v  validator.Validator
}

func NewPricePerformanceHandler(uc usecase.PricePerformanceUsecase, v validator.Validator) *PricePerformanceHandler {
	return &PricePerformanceHandler{uc: uc, v: v}
}

type pricePerformanceRequest struct {
	Symbol string `json:"symbol" validate:"required"`
}

type pricePerformanceResponse struct {
	Prices []priceResponse `json:"prices"`
}

type priceResponse struct {
	Close      rawFormattedResponse `json:"close"`
	High       rawFormattedResponse `json:"high"`
	Low        rawFormattedResponse `json:"low"`
	Percentage percentResponse      `json:"percentage"`
	Timeframe  string               `json:"timeframe"`
}

type rawFormattedResponse struct {
	Raw       float64 `json:"raw"`
	Formatted string  `json:"formatted"`
}

type percentResponse struct {
	Raw       float64 `json:"raw"`
	Formatted string  `json:"formatted"`
}

func (h *PricePerformanceHandler) PricePerformance(w http.ResponseWriter, r *http.Request) {
	req := pricePerformanceRequest{Symbol: chi.URLParam(r, "symbol")}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate price performance params")
		return
	}

	data, err := h.uc.GetPricePerformance(r.Context(), req.Symbol)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get price performance")
		return
	}
	response.OK(w, toResponse(data))
}

func toResponse(d *domain.PricePerformanceData) pricePerformanceResponse {
	out := pricePerformanceResponse{Prices: make([]priceResponse, 0, len(d.Prices))}
	for _, p := range d.Prices {
		out.Prices = append(out.Prices, priceResponse{
			Close:      rawFormattedResponse{Raw: p.Close.Raw, Formatted: p.Close.Formatted},
			High:       rawFormattedResponse{Raw: p.High.Raw, Formatted: p.High.Formatted},
			Low:        rawFormattedResponse{Raw: p.Low.Raw, Formatted: p.Low.Formatted},
			Percentage: percentResponse{Raw: p.Percentage.Raw, Formatted: p.Percentage.Formatted},
			Timeframe:  p.Timeframe,
		})
	}
	return out
}
