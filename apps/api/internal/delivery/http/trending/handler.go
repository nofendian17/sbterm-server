package trending

import (
	"net/http"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
)

type TrendingHandler struct {
	uc usecase.TrendingUsecase
}

func NewTrendingHandler(uc usecase.TrendingUsecase) *TrendingHandler {
	return &TrendingHandler{uc: uc}
}

type trendingStockResponse struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Last     string `json:"last"`
	Change   string `json:"change"`
	Percent  string `json:"percent"`
	Previous string `json:"previous"`
	LogoURL  string `json:"logo"`
	Status   string `json:"status"`
}

func (h *TrendingHandler) Trending(w http.ResponseWriter, r *http.Request) {
	stocks, err := h.uc.GetTrending(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get trending stocks")
		return
	}
	response.OK(w, toResponses(stocks))
}

func toResponses(stocks []domain.TrendingStock) []trendingStockResponse {
	res := make([]trendingStockResponse, 0, len(stocks))
	for _, s := range stocks {
		res = append(res, trendingStockResponse{
			Symbol:   s.Symbol,
			Name:     s.Name,
			Last:     s.Last,
			Change:   s.Change,
			Percent:  s.Percent,
			Previous: s.Previous,
			LogoURL:  s.LogoURL,
			Status:   s.Status,
		})
	}
	return res
}
