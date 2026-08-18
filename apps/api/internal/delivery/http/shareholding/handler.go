package shareholding

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

type ShareholdingHandler struct {
	uc usecase.ShareholdingCompositionUsecase
	v  validator.Validator
}

func NewShareholdingHandler(uc usecase.ShareholdingCompositionUsecase, v validator.Validator) *ShareholdingHandler {
	return &ShareholdingHandler{uc: uc, v: v}
}

type shareholdingRequest struct {
	Symbol      string `json:"symbol" validate:"required"`
	PeriodStart string `json:"period_start" validate:"omitempty"`
	PeriodEnd   string `json:"period_end" validate:"omitempty"`
}

type compositionPeriodResponse struct {
	ReportDate   string                `json:"report_date"`
	TotalShares  rawFormattedResponse  `json:"total_shares"`
	Compositions []compositionResponse `json:"compositions"`
}

type rawFormattedResponse struct {
	Raw       string `json:"raw"`
	Formatted string `json:"formatted"`
}

type compositionResponse struct {
	Label      string               `json:"label"`
	Shares     rawFormattedResponse `json:"shares"`
	Percentage percentResponse      `json:"percentage"`
	Colors     colorsResponse       `json:"colors"`
}

type percentResponse struct {
	Raw       float64 `json:"raw"`
	Formatted string  `json:"formatted"`
}

type colorsResponse struct {
	Light string `json:"light"`
	Dark  string `json:"dark"`
}

func (h *ShareholdingHandler) ShareholdingComposition(w http.ResponseWriter, r *http.Request) {
	req := shareholdingRequest{
		Symbol:      chi.URLParam(r, "symbol"),
		PeriodStart: r.URL.Query().Get("period_start"),
		PeriodEnd:   r.URL.Query().Get("period_end"),
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate shareholding params")
		return
	}

	periods, err := h.uc.GetShareholdingComposition(r.Context(), req.Symbol, req.PeriodStart, req.PeriodEnd)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get shareholding composition")
		return
	}
	response.OK(w, toResponses(periods))
}

func toResponses(periods []domain.ShareholdingCompositionPeriod) []compositionPeriodResponse {
	res := make([]compositionPeriodResponse, 0, len(periods))
	for _, p := range periods {
		comps := make([]compositionResponse, 0, len(p.Compositions))
		for _, c := range p.Compositions {
			comps = append(comps, compositionResponse{
				Label:      c.Label,
				Shares:     rawFormattedResponse{Raw: c.Shares.Raw, Formatted: c.Shares.Formatted},
				Percentage: percentResponse{Raw: c.Percentage.Raw, Formatted: c.Percentage.Formatted},
				Colors:     colorsResponse{Light: c.Colors.Light, Dark: c.Colors.Dark},
			})
		}
		res = append(res, compositionPeriodResponse{
			ReportDate:   p.ReportDate,
			TotalShares:  rawFormattedResponse{Raw: p.TotalShares.Raw, Formatted: p.TotalShares.Formatted},
			Compositions: comps,
		})
	}
	return res
}
