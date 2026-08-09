package sectors

import (
	"net/http"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/response"
)

type SectorsHandler struct {
	uc usecase.SectorsUsecase
}

func NewSectorsHandler(uc usecase.SectorsUsecase) *SectorsHandler {
	return &SectorsHandler{uc: uc}
}

type sectorResponse struct {
	Symbol    string            `json:"symbol"`
	Icon      string            `json:"icon"`
	Type      string            `json:"type"`
	Last      float64           `json:"last"`
	Change    string            `json:"change"`
	Percent   float64           `json:"percent"`
	Companies []companyResponse `json:"companies"`
}

type companyResponse struct {
	Symbol        string `json:"symbol"`
	Name          string `json:"name"`
	Last          string `json:"last"`
	Change        string `json:"change"`
	Percent       string `json:"percent"`
	Volume        int64  `json:"volume"`
	Value         int64  `json:"value"`
	MarketCap     string `json:"marketcap"`
	IconURL       string `json:"icon_url"`
	CompanyStatus string `json:"company_status"`
	IsUMA         bool   `json:"is_uma"`
}

func (h *SectorsHandler) Sectors(w http.ResponseWriter, r *http.Request) {
	sectors, err := h.uc.GetSectors(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get sectors")
		return
	}
	response.OK(w, toResponses(sectors))
}

func toResponses(sectors []domain.Sector) []sectorResponse {
	res := make([]sectorResponse, 0, len(sectors))
	for _, s := range sectors {
		companies := make([]companyResponse, 0, len(s.Companies))
		for _, c := range s.Companies {
			companies = append(companies, companyResponse{
				Symbol:        c.Symbol,
				Name:          c.Name,
				Last:          c.Last,
				Change:        c.Change,
				Percent:       c.Percent,
				Volume:        c.Volume,
				Value:         c.Value,
				MarketCap:     c.MarketCap,
				IconURL:       c.IconURL,
				CompanyStatus: c.CompanyStatus,
				IsUMA:         c.IsUMA,
			})
		}
		res = append(res, sectorResponse{
			Symbol:    s.Symbol,
			Icon:      s.Icon,
			Type:      s.Type,
			Last:      s.Last,
			Change:    s.Change,
			Percent:   s.Percent,
			Companies: companies,
		})
	}
	return res
}
