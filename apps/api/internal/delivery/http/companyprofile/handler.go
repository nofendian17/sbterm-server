package companyprofile

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

type CompanyProfileHandler struct {
	uc usecase.CompanyProfileUsecase
	v  validator.Validator
}

func NewCompanyProfileHandler(uc usecase.CompanyProfileUsecase, v validator.Validator) *CompanyProfileHandler {
	return &CompanyProfileHandler{uc: uc, v: v}
}

type companyProfileRequest struct {
	Symbol string `json:"symbol" validate:"required"`
}

type companyProfileResponse struct {
	Background                      string                      `json:"background"`
	History                         *historyResponse            `json:"history"`
	KeyExecutive                    *keyExecutiveResponse       `json:"key_executive"`
	Address                         []addressResponse           `json:"address"`
	Subsidiary                      []subsidiaryResponse        `json:"subsidiary"`
	Beneficiary                     []beneficiaryResponse       `json:"beneficiary"`
	Shareholder                     []shareholderResponse       `json:"shareholder"`
	ShareholderDirectorCommissioner []shareholderResponse       `json:"shareholder_director_commissioner"`
	ShareholderNumbers              []shareholderNumberResponse `json:"shareholder_numbers"`
	ShareholderOnePercent           []shareholderIDResponse     `json:"shareholder_one_percent"`
}

type shareholderResponse struct {
	Percentage string   `json:"percentage"`
	Name       string   `json:"name"`
	Value      string   `json:"value"`
	Badges     []string `json:"badges"`
}

// shareholderIDResponse is the full shareholder listing shape used for
// shareholder_one_percent, whose entries carry meaningful ids and metadata.
// ponytail: if the other lists ever carry this data, fold into
// shareholderResponse and drop this type.
type shareholderIDResponse struct {
	ID             string   `json:"id"`
	Percentage     string   `json:"percentage"`
	Name           string   `json:"name"`
	Value          string   `json:"value"`
	Badges         []string `json:"badges"`
	Type           string   `json:"type"`
	Location       string   `json:"location"`
	Nationality    string   `json:"nationality"`
	Domicile       string   `json:"domicile"`
	Scripless      string   `json:"scripless"`
	Scrip          string   `json:"scrip"`
	ValueFormatted string   `json:"value_formatted"`
	Classification string   `json:"classification"`
}

type shareholderNumberResponse struct {
	ShareholderDate string `json:"shareholder_date"`
	TotalShare      string `json:"total_share"`
	Change          int64  `json:"change"`
	ChangeFormatted string `json:"change_formatted"`
}

type historyResponse struct {
	Amount       string   `json:"amount"`
	Board        string   `json:"board"`
	Date         string   `json:"date"`
	Price        string   `json:"price"`
	Registrar    string   `json:"registrar"`
	Shares       string   `json:"shares"`
	Underwriters []string `json:"underwriters"`
	FreeFloat    string   `json:"free_float"`
}

type keyExecutiveResponse struct {
	Commissioner            []executiveResponse `json:"commissioner"`
	Director                []executiveResponse `json:"director"`
	IndependentCommissioner []executiveResponse `json:"independent_commissioner"`
}

type executiveResponse struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type addressResponse struct {
	Office  string   `json:"office"`
	Phone   string   `json:"phone"`
	Fax     string   `json:"fax"`
	Email   []string `json:"email"`
	Website string   `json:"website"`
	NPWP    string   `json:"npwp"`
}

type subsidiaryResponse struct {
	Company    string `json:"company"`
	Percentage string `json:"percentage"`
	Types      string `json:"types"`
	Value      string `json:"value"`
}

type beneficiaryResponse struct {
	Name string `json:"name"`
}

func (h *CompanyProfileHandler) CompanyProfile(w http.ResponseWriter, r *http.Request) {
	req := companyProfileRequest{Symbol: chi.URLParam(r, "symbol")}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate company profile params")
		return
	}

	profile, err := h.uc.GetProfile(r.Context(), req.Symbol)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get company profile")
		return
	}
	response.OK(w, toResponse(profile))
}

func toResponse(p *domain.CompanyProfile) companyProfileResponse {
	out := companyProfileResponse{
		Background:                      p.Background,
		Address:                         make([]addressResponse, 0, len(p.Address)),
		Subsidiary:                      make([]subsidiaryResponse, 0, len(p.Subsidiary)),
		Beneficiary:                     make([]beneficiaryResponse, 0, len(p.Beneficiary)),
		Shareholder:                     toShareholderResponses(p.Shareholder),
		ShareholderDirectorCommissioner: toShareholderResponses(p.ShareholderDirectorCommissioner),
		ShareholderNumbers:              toShareholderNumberResponses(p.ShareholderNumbers),
		ShareholderOnePercent:           toShareholderIDResponses(p.ShareholderOnePercent),
	}
	if p.History != nil {
		out.History = &historyResponse{
			Amount:       p.History.Amount,
			Board:        p.History.Board,
			Date:         p.History.Date,
			Price:        p.History.Price,
			Registrar:    p.History.Registrar,
			Shares:       p.History.Shares,
			Underwriters: p.History.Underwriters,
			FreeFloat:    p.History.FreeFloat,
		}
	}
	if p.KeyExecutive != nil {
		out.KeyExecutive = &keyExecutiveResponse{
			Commissioner:            toExecutiveResponses(p.KeyExecutive.Commissioner),
			Director:                toExecutiveResponses(p.KeyExecutive.Director),
			IndependentCommissioner: toExecutiveResponses(p.KeyExecutive.IndependentCommissioner),
		}
	}
	for _, a := range p.Address {
		out.Address = append(out.Address, addressResponse{
			Office:  a.Office,
			Phone:   a.Phone,
			Fax:     a.Fax,
			Email:   a.Email,
			Website: a.Website,
			NPWP:    a.NPWP,
		})
	}
	for _, s := range p.Subsidiary {
		out.Subsidiary = append(out.Subsidiary, subsidiaryResponse{
			Company:    s.Company,
			Percentage: s.Percentage,
			Types:      s.Types,
			Value:      s.Value,
		})
	}
	for _, b := range p.Beneficiary {
		out.Beneficiary = append(out.Beneficiary, beneficiaryResponse{Name: b.Name})
	}
	return out
}

func toExecutiveResponses(in []domain.ProfileExecutive) []executiveResponse {
	out := make([]executiveResponse, 0, len(in))
	for _, e := range in {
		out = append(out, executiveResponse{ID: e.ID, Key: e.Key, Value: e.Value})
	}
	return out
}

func toShareholderResponses(in []domain.ProfileShareholder) []shareholderResponse {
	out := make([]shareholderResponse, 0, len(in))
	for _, s := range in {
		out = append(out, shareholderResponse{
			Percentage: s.Percentage,
			Name:       s.Name,
			Value:      s.Value,
			Badges:     s.Badges,
		})
	}
	return out
}

func toShareholderIDResponses(in []domain.ProfileShareholder) []shareholderIDResponse {
	out := make([]shareholderIDResponse, 0, len(in))
	for _, s := range in {
		out = append(out, shareholderIDResponse{
			ID:             s.ID,
			Percentage:     s.Percentage,
			Name:           s.Name,
			Value:          s.Value,
			Badges:         s.Badges,
			Type:           s.Type,
			Location:       s.Location,
			Nationality:    s.Nationality,
			Domicile:       s.Domicile,
			Scripless:      s.Scripless,
			Scrip:          s.Scrip,
			ValueFormatted: s.ValueFormatted,
			Classification: s.Classification,
		})
	}
	return out
}

func toShareholderNumberResponses(in []domain.ProfileShareholderNumber) []shareholderNumberResponse {
	out := make([]shareholderNumberResponse, 0, len(in))
	for _, n := range in {
		out = append(out, shareholderNumberResponse{
			ShareholderDate: n.ShareholderDate,
			TotalShare:      n.TotalShare,
			Change:          n.Change,
			ChangeFormatted: n.ChangeFormatted,
		})
	}
	return out
}
