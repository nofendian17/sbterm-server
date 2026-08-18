package mover

import (
	"net/http"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

type MarketMoverHandler struct {
	uc usecase.MarketMoverUsecase
	v  validator.Validator
}

func NewMarketMoverHandler(uc usecase.MarketMoverUsecase, v validator.Validator) *MarketMoverHandler {
	return &MarketMoverHandler{uc: uc, v: v}
}

type marketMoverRequest struct {
	MoverType    string   `json:"mover_type" validate:"required,oneof=MOVER_TYPE_TOP_GAINER MOVER_TYPE_TOP_LOSER MOVER_TYPE_TOP_VALUE MOVER_TYPE_TOP_VOLUME MOVER_TYPE_TOP_FREQUENCY MOVER_TYPE_NET_FOREIGN_BUY MOVER_TYPE_NET_FOREIGN_SELL MOVER_TYPE_IEVAL_TOP_GAINER"`
	FilterStocks []string `json:"filter_stocks" validate:"omitempty,dive,oneof=FILTER_STOCKS_TYPE_MAIN_BOARD FILTER_STOCKS_TYPE_DEVELOPMENT_BOARD FILTER_STOCKS_TYPE_ACCELERATION_BOARD FILTER_STOCKS_TYPE_NEW_ECONOMY_BOARD FILTER_STOCKS_TYPE_SPECIAL_MONITORING_BOARD FILTER_STOCKS_TYPE_WARRANT_AND_RIGHT"`
}

type marketMoverResponse struct {
	Symbol         string  `json:"symbol"`
	Name           string  `json:"name"`
	Price          float64 `json:"price"`
	ChangeValue    float64 `json:"change_value"`
	ChangePercent  float64 `json:"change_percent"`
	Value          float64 `json:"value"`
	Volume         float64 `json:"volume"`
	Frequency      float64 `json:"freq"`
	NetForeignBuy  float64 `json:"net_foreign_buy"`
	NetForeignSell float64 `json:"net_foreign_sell"`
	IEP            float64 `json:"iep"`
	IEV            float64 `json:"iev"`
	IEVAL          float64 `json:"ieval"`
	IEPChangePrev  float64 `json:"iep_change_prev"`
}

func (h *MarketMoverHandler) MarketMover(w http.ResponseWriter, r *http.Request) {
	req := marketMoverRequest{
		MoverType:    r.URL.Query().Get("mover_type"),
		FilterStocks: r.URL.Query()["filter_stocks"],
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate market mover params")
		return
	}

	movers, err := h.uc.GetMarketMover(r.Context(), req.MoverType, req.FilterStocks)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get market movers")
		return
	}
	response.OK(w, toResponses(movers))
}

func toResponses(movers []domain.MarketMover) []marketMoverResponse {
	res := make([]marketMoverResponse, 0, len(movers))
	for _, m := range movers {
		res = append(res, marketMoverResponse{
			Symbol:         m.Symbol,
			Name:           m.Name,
			Price:          m.Price,
			ChangeValue:    m.ChangeValue,
			ChangePercent:  m.ChangePercent,
			Value:          m.Value,
			Volume:         m.Volume,
			Frequency:      m.Frequency,
			NetForeignBuy:  m.NetForeignBuy,
			NetForeignSell: m.NetForeignSell,
			IEP:            m.IEP,
			IEV:            m.IEV,
			IEVAL:          m.IEVAL,
			IEPChangePrev:  m.IEPChangePrev,
		})
	}
	return res
}
