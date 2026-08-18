package subsidiary

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

type SubsidiaryHandler struct {
	uc usecase.SubsidiaryUsecase
	v  validator.Validator
}

func NewSubsidiaryHandler(uc usecase.SubsidiaryUsecase, v validator.Validator) *SubsidiaryHandler {
	return &SubsidiaryHandler{uc: uc, v: v}
}

type subsidiaryRequest struct {
	Symbol string `json:"symbol" validate:"required"`
}

type subsidiaryResponse struct {
	Currency          string                   `json:"currency"`
	LastUpdatedPeriod string                   `json:"last_updated_period"`
	Unit              string                   `json:"unit"`
	Subsidiaries      []subsidiaryItemResponse `json:"subsidiaries"`
}

type subsidiaryItemResponse struct {
	CompanyName       string  `json:"company_name"`
	BusinessType      string  `json:"business_type"`
	Location          string  `json:"location"`
	CommercialYear    string  `json:"commercial_year"`
	TotalAssets       string  `json:"total_assets"`
	Percentage        string  `json:"percentage"`
	OperationalStatus string  `json:"operational_status"`
	Period            string  `json:"period"`
	Raw               *string `json:"raw"`
}

func (h *SubsidiaryHandler) Subsidiaries(w http.ResponseWriter, r *http.Request) {
	req := subsidiaryRequest{Symbol: chi.URLParam(r, "symbol")}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate subsidiary params")
		return
	}

	subsidiaries, err := h.uc.GetSubsidiaries(r.Context(), req.Symbol)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get subsidiaries")
		return
	}
	response.OK(w, toResponse(subsidiaries))
}

func toResponse(d *domain.SubsidiaryData) subsidiaryResponse {
	res := subsidiaryResponse{
		Currency:          d.Currency,
		LastUpdatedPeriod: d.LastUpdatedPeriod,
		Unit:              d.Unit,
		Subsidiaries:      make([]subsidiaryItemResponse, 0, len(d.Subsidiaries)),
	}
	for _, s := range d.Subsidiaries {
		res.Subsidiaries = append(res.Subsidiaries, subsidiaryItemResponse{
			CompanyName:       s.CompanyName,
			BusinessType:      s.BusinessType,
			Location:          s.Location,
			CommercialYear:    s.CommercialYear,
			TotalAssets:       s.TotalAssets,
			Percentage:        s.Percentage,
			OperationalStatus: s.OperationalStatus,
			Period:            s.Period,
			Raw:               s.Raw,
		})
	}
	return res
}
