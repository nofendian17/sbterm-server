package stocks

import (
	"net/http"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/response"
)

type StocksHandler struct {
	uc usecase.StocksUsecase
}

func NewStocksHandler(uc usecase.StocksUsecase) *StocksHandler {
	return &StocksHandler{uc: uc}
}

type stockResponse struct {
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

func (h *StocksHandler) Stocks(w http.ResponseWriter, r *http.Request) {
	stocks, err := h.uc.GetStocks(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get stocks")
		return
	}
	response.OK(w, toResponses(stocks))
}

func toResponses(stocks []domain.Stock) []stockResponse {
	res := make([]stockResponse, 0, len(stocks))
	for _, s := range stocks {
		res = append(res, stockResponse{
			Symbol:        s.Symbol,
			Name:          s.Name,
			Last:          s.Last,
			Change:        s.Change,
			Percent:       s.Percent,
			Volume:        s.Volume,
			Value:         s.Value,
			MarketCap:     s.MarketCap,
			IconURL:       s.IconURL,
			CompanyStatus: s.CompanyStatus,
			IsUMA:         s.IsUMA,
		})
	}
	return res
}