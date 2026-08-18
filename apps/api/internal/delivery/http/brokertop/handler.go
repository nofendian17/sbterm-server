package brokertop

import (
	"net/http"
	"strconv"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

const (
	defaultSort       = "TB_SORT_BY_TOTAL_VALUE"
	defaultOrder      = "ORDER_BY_DESC"
	defaultPeriod     = "TB_PERIOD_LAST_1_DAY"
	defaultMarketType = "MARKET_TYPE_ALL"
	defaultEodOnly    = true
)

type BrokerTopHandler struct {
	uc usecase.BrokerTopUsecase
	v  validator.Validator
}

func NewBrokerTopHandler(uc usecase.BrokerTopUsecase, v validator.Validator) *BrokerTopHandler {
	return &BrokerTopHandler{uc: uc, v: v}
}

type brokerTopRequest struct {
	Sort       string `json:"sort" validate:"omitempty,oneof=TB_SORT_BY_TOTAL_VALUE TB_SORT_BY_NET_VALUE TB_SORT_BY_BUY_VALUE TB_SORT_BY_SELL_VALUE TB_SORT_BY_TOTAL_FREQUENCY"`
	Order      string `json:"order" validate:"omitempty,oneof=ORDER_BY_ASC ORDER_BY_DESC"`
	Period     string `json:"period" validate:"omitempty,oneof=TB_PERIOD_LAST_1_DAY TB_PERIOD_LAST_7_DAYS TB_PERIOD_LAST_1_MONTH TB_PERIOD_YEAR_TO_DATE"`
	MarketType string `json:"market_type" validate:"omitempty,oneof=MARKET_TYPE_ALL"`
	EodOnly    bool   `json:"eod_only"`
}

type brokerTopResponse struct {
	Date brokerTopDateResp   `json:"date"`
	List []brokerTopItemResp `json:"list"`
}

type brokerTopDateResp struct {
	From string `json:"from"`
	To   string `json:"to"`
	Idx  string `json:"idx"`
}

type brokerTopItemResp struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	InvestorType   string `json:"investor_type"`
	TotalValue     string `json:"total_value"`
	NetValue       string `json:"net_value"`
	BuyValue       string `json:"buy_value"`
	SellValue      string `json:"sell_value"`
	TotalVolume    string `json:"total_volume"`
	TotalFrequency string `json:"total_frequency"`
	Group          string `json:"group"`
}

func parseBoolQuery(s string, def bool) (bool, bool) {
	if s == "" {
		return def, true
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return false, false
	}
	return b, true
}

func (h *BrokerTopHandler) BrokerTop(w http.ResponseWriter, r *http.Request) {
	eodOnly, ok := parseBoolQuery(r.URL.Query().Get("eod_only"), defaultEodOnly)
	if !ok {
		response.ValidationError(w, "validation failed", map[string]string{"eod_only": "must be a valid boolean"})
		return
	}

	req := brokerTopRequest{
		Sort:       r.URL.Query().Get("sort"),
		Order:      r.URL.Query().Get("order"),
		Period:     r.URL.Query().Get("period"),
		MarketType: r.URL.Query().Get("market_type"),
		EodOnly:    eodOnly,
	}
	if req.Sort == "" {
		req.Sort = defaultSort
	}
	if req.Order == "" {
		req.Order = defaultOrder
	}
	if req.Period == "" {
		req.Period = defaultPeriod
	}
	if req.MarketType == "" {
		req.MarketType = defaultMarketType
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate broker top params")
		return
	}

	data, err := h.uc.GetBrokerTop(r.Context(), req.Sort, req.Order, req.Period, req.MarketType, req.EodOnly)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get broker top")
		return
	}
	response.OK(w, toResponse(data))
}

func toResponse(d *domain.BrokerTopData) brokerTopResponse {
	out := brokerTopResponse{
		Date: brokerTopDateResp{From: d.Date.From, To: d.Date.To, Idx: d.Date.Idx},
		List: make([]brokerTopItemResp, 0, len(d.List)),
	}
	for _, it := range d.List {
		out.List = append(out.List, brokerTopItemResp{
			Code:           it.Code,
			Name:           it.Name,
			InvestorType:   it.InvestorType,
			TotalValue:     it.TotalValue,
			NetValue:       it.NetValue,
			BuyValue:       it.BuyValue,
			SellValue:      it.SellValue,
			TotalVolume:    it.TotalVolume,
			TotalFrequency: it.TotalFrequency,
			Group:          it.Group,
		})
	}
	return out
}
