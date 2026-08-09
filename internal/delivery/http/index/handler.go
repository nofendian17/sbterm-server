package index

import (
	"net/http"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/response"
)

type IndexHandler struct {
	uc usecase.IndexUsecase
}

func NewIndexHandler(uc usecase.IndexUsecase) *IndexHandler {
	return &IndexHandler{uc: uc}
}

type indexesResponse struct {
	Main []indexResponse `json:"main"`
	All  []indexResponse `json:"all"`
}

type indexResponse struct {
	Symbol    string `json:"symbol"`
	Name      string `json:"name"`
	Last      string `json:"last"`
	Change    string `json:"change"`
	Percent   string `json:"percent"`
	MarketCap string `json:"marketcap"`
}

func (h *IndexHandler) Index(w http.ResponseWriter, r *http.Request) {
	indexes, err := h.uc.GetIndexes(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get market indexes")
		return
	}
	response.OK(w, indexesResponse{
		Main: toResponses(indexes.Main),
		All:  toResponses(indexes.All),
	})
}

func toResponses(list []domain.Index) []indexResponse {
	res := make([]indexResponse, 0, len(list))
	for _, i := range list {
		res = append(res, indexResponse{
			Symbol:    i.Symbol,
			Name:      i.Name,
			Last:      i.Last,
			Change:    i.Change,
			Percent:   i.Percent,
			MarketCap: i.MarketCap,
		})
	}
	return res
}
