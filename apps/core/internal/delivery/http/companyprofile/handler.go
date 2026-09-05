// Package http provides HTTP handlers for the core service API.

package companyprofile

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

// CompanyProfileHandler serves the per-stock company profile aggregate.
type CompanyProfileHandler struct {
	uc usecase.CompanyProfileUsecase
	v  validator.Validator
}

func NewCompanyProfileHandler(uc usecase.CompanyProfileUsecase, v validator.Validator) *CompanyProfileHandler {
	return &CompanyProfileHandler{uc: uc, v: v}
}

type executiveDTO struct {
	Kind       string  `json:"kind"`
	Name       string  `json:"name"`
	Role       *string `json:"role,omitempty"`
	ExternalID *string `json:"external_id,omitempty"`
	Position   int     `json:"position,omitempty"`
}

type holdingDTO struct {
	HolderGroup   string   `json:"holder_group"`
	Name          string   `json:"name"`
	Percentage    *float64 `json:"percentage,omitempty"`
	PercentageRaw *string  `json:"percentage_raw,omitempty"`
	AmountRaw     *string  `json:"amount_raw,omitempty"`
	Badges        []string `json:"badges,omitempty"`
	Position      int      `json:"position,omitempty"`
}

type shareholderNumberDTO struct {
	ShareholderDate string  `json:"shareholder_date"`
	TotalShare      *string `json:"total_share,omitempty"`
	Change          *int64  `json:"change,omitempty"`
	ChangeFormatted *string `json:"change_formatted,omitempty"`
}

type subsidiaryDTO struct {
	Name              string   `json:"name"`
	BusinessType      *string  `json:"business_type,omitempty"`
	Location          *string  `json:"location,omitempty"`
	CommercialYear    *string  `json:"commercial_year,omitempty"`
	TotalAssets       *float64 `json:"total_assets,omitempty"`
	TotalAssetsRaw    *string  `json:"total_assets_raw,omitempty"`
	Percentage        *float64 `json:"percentage,omitempty"`
	PercentageRaw     *string  `json:"percentage_raw,omitempty"`
	OperationalStatus *string  `json:"operational_status,omitempty"`
	Period            *string  `json:"period,omitempty"`
	Position          int      `json:"position,omitempty"`
}

type addressDTO struct {
	Office   *string  `json:"office,omitempty"`
	Phone    *string  `json:"phone,omitempty"`
	Fax      *string  `json:"fax,omitempty"`
	Website  *string  `json:"website,omitempty"`
	Npwp     *string  `json:"npwp,omitempty"`
	Emails   []string `json:"emails,omitempty"`
	Position int      `json:"position,omitempty"`
}

type profileDTO struct {
	Symbol             string                 `json:"symbol"`
	Background         *string                `json:"background,omitempty"`
	Board              *string                `json:"board,omitempty"`
	ListingDate        *string                `json:"listing_date,omitempty"`
	ListingPrice       *string                `json:"listing_price,omitempty"`
	IpoAmount          *string                `json:"ipo_amount,omitempty"`
	ListedShares       *string                `json:"listed_shares,omitempty"`
	FreeFloat          *string                `json:"free_float,omitempty"`
	Registrar          *string                `json:"registrar,omitempty"`
	Executives         []executiveDTO         `json:"executives,omitempty"`
	Holdings           []holdingDTO           `json:"holdings,omitempty"`
	ShareholderNumbers []shareholderNumberDTO `json:"shareholder_numbers,omitempty"`
	Subsidiaries       []subsidiaryDTO        `json:"subsidiaries,omitempty"`
	Addresses          []addressDTO           `json:"addresses,omitempty"`
}

func toDTO(p domain.CompanyProfile) profileDTO {
	out := profileDTO{
		Symbol:       p.Symbol,
		Background:   p.Background,
		Board:        p.Board,
		ListingDate:  p.ListingDate,
		ListingPrice: p.ListingPrice,
		IpoAmount:    p.IpoAmount,
		ListedShares: p.ListedShares,
		FreeFloat:    p.FreeFloat,
		Registrar:    p.Registrar,
	}
	for _, e := range p.Executives {
		out.Executives = append(out.Executives, executiveDTO{
			Kind: e.Kind, Name: e.Name, Role: e.Role,
			ExternalID: e.ExternalID, Position: e.Position,
		})
	}
	for _, h := range p.Holdings {
		out.Holdings = append(out.Holdings, holdingDTO{
			HolderGroup: h.HolderGroup, Name: h.Name, Percentage: h.Percentage,
			PercentageRaw: h.PercentageRaw, AmountRaw: h.AmountRaw,
			Badges: h.Badges, Position: h.Position,
		})
	}
	for _, n := range p.ShareholderNumbers {
		out.ShareholderNumbers = append(out.ShareholderNumbers, shareholderNumberDTO{
			ShareholderDate: n.ShareholderDate, TotalShare: n.TotalShare,
			Change: n.Change, ChangeFormatted: n.ChangeFormatted,
		})
	}
	for _, s := range p.Subsidiaries {
		out.Subsidiaries = append(out.Subsidiaries, subsidiaryDTO{
			Name: s.Name, BusinessType: s.BusinessType, Location: s.Location,
			CommercialYear: s.CommercialYear, TotalAssets: s.TotalAssets,
			TotalAssetsRaw: s.TotalAssetsRaw, Percentage: s.Percentage,
			PercentageRaw: s.PercentageRaw, OperationalStatus: s.OperationalStatus,
			Period: s.Period, Position: s.Position,
		})
	}
	for _, a := range p.Addresses {
		out.Addresses = append(out.Addresses, addressDTO{
			Office: a.Office, Phone: a.Phone, Fax: a.Fax, Website: a.Website,
			Npwp: a.Npwp, Emails: a.Emails, Position: a.Position,
		})
	}
	return out
}

func (d profileDTO) toDomain() domain.CompanyProfile {
	p := domain.CompanyProfile{
		Symbol:       d.Symbol,
		Background:   d.Background,
		Board:        d.Board,
		ListingDate:  d.ListingDate,
		ListingPrice: d.ListingPrice,
		IpoAmount:    d.IpoAmount,
		ListedShares: d.ListedShares,
		FreeFloat:    d.FreeFloat,
		Registrar:    d.Registrar,
	}
	for _, e := range d.Executives {
		p.Executives = append(p.Executives, domain.CompanyExecutive{
			Kind: e.Kind, Name: e.Name, Role: e.Role,
			ExternalID: e.ExternalID, Position: e.Position,
		})
	}
	for _, h := range d.Holdings {
		p.Holdings = append(p.Holdings, domain.CompanyHolding{
			HolderGroup: h.HolderGroup, Name: h.Name, Percentage: h.Percentage,
			PercentageRaw: h.PercentageRaw, AmountRaw: h.AmountRaw,
			Badges: h.Badges, Position: h.Position,
		})
	}
	for _, n := range d.ShareholderNumbers {
		p.ShareholderNumbers = append(p.ShareholderNumbers, domain.CompanyShareholderNumber{
			ShareholderDate: n.ShareholderDate, TotalShare: n.TotalShare,
			Change: n.Change, ChangeFormatted: n.ChangeFormatted,
		})
	}
	for _, s := range d.Subsidiaries {
		p.Subsidiaries = append(p.Subsidiaries, domain.CompanySubsidiary{
			Name: s.Name, BusinessType: s.BusinessType, Location: s.Location,
			CommercialYear: s.CommercialYear, TotalAssets: s.TotalAssets,
			TotalAssetsRaw: s.TotalAssetsRaw, Percentage: s.Percentage,
			PercentageRaw: s.PercentageRaw, OperationalStatus: s.OperationalStatus,
			Period: s.Period, Position: s.Position,
		})
	}
	for _, a := range d.Addresses {
		p.Addresses = append(p.Addresses, domain.CompanyAddress{
			Office: a.Office, Phone: a.Phone, Fax: a.Fax, Website: a.Website,
			Npwp: a.Npwp, Emails: a.Emails, Position: a.Position,
		})
	}
	return p
}

// Sync fetches the symbol's profile from the configured upstream and
// replaces the local cluster (admin, stocks:sync).
func (h *CompanyProfileHandler) Sync(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "symbol is required")
		return
	}
	p, err := h.uc.SyncProfile(r.Context(), symbol)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidInput):
			response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "validation failed")
		case errors.Is(err, domain.ErrStockNotFound):
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "stock not found")
		case errors.Is(err, domain.ErrCompanyProfileSyncFailed):
			response.Error(w, http.StatusBadGateway, response.CodeUpstreamError, "company profile sync failed")
		default:
			response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		}
		return
	}
	response.OK(w, toDTO(p))
}

// mapProfileError maps domain errors to HTTP responses for admin profile ops.
func mapProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "validation failed")
	case errors.Is(err, domain.ErrStockNotFound):
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "stock not found")
	case errors.Is(err, domain.ErrCompanyProfileSyncFailed):
		response.Error(w, http.StatusBadGateway, response.CodeUpstreamError, "company profile sync failed")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
	}
}

// Get returns the aggregate profile for a symbol (user-facing, stocks:read).
func (h *CompanyProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "symbol is required")
		return
	}
	p, err := h.uc.Get(r.Context(), symbol)
	if err != nil {
		if errors.Is(err, domain.ErrCompanyProfileNotFound) {
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "company profile not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	response.OK(w, toDTO(p))
}

// Save creates or replaces the whole profile cluster (admin, stocks:write).
func (h *CompanyProfileHandler) Save(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "symbol is required")
		return
	}
	var body profileDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	if body.Symbol != "" && body.Symbol != symbol {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "symbol mismatch between path and body")
		return
	}
	p := body.toDomain()
	p.Symbol = symbol

	if err := h.uc.Save(r.Context(), p); err != nil {
		mapProfileError(w, err)
		return
	}
	response.Message(w, http.StatusOK, "company profile saved")
}
