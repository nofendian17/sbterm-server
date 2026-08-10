package topstock

import (
	"net/http"
	"strconv"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/response"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

// defaultFilter values applied when the corresponding query param is omitted.
const (
	defaultInvestorType = "INVESTOR_TYPE_ALL"
	defaultMarketType   = "MARKET_TYPE_ALL"
	defaultValueType    = "VALUE_TYPE_NET"
)

type TopStockHandler struct {
	uc usecase.TopStockUsecase
	v  validator.Validator
}

func NewTopStockHandler(uc usecase.TopStockUsecase, v validator.Validator) *TopStockHandler {
	return &TopStockHandler{uc: uc, v: v}
}

type topStockRequest struct {
	Start        string `json:"start" validate:"required,datetime=2006-01-02"`
	End          string `json:"end" validate:"required,datetime=2006-01-02"`
	InvestorType string `json:"investor_type" validate:"omitempty,oneof=INVESTOR_TYPE_ALL INVESTOR_TYPE_FOREIGN INVESTOR_TYPE_DOMESTIC"`
	MarketType   string `json:"market_type" validate:"omitempty,oneof=MARKET_TYPE_ALL MARKET_TYPE_REGULER MARKET_TYPE_TUNAI MARKET_TYPE_NEGO"`
	ValueType    string `json:"value_type" validate:"omitempty,oneof=VALUE_TYPE_NET VALUE_TYPE_GROSS VALUE_TYPE_TOTAL"`
	Page         int    `json:"page" validate:"omitempty"`
}

type topStockResponse struct {
	TopBuy        []topStockItemResponse `json:"top_buy"`
	TopSell       []topStockItemResponse `json:"top_sell"`
	Total         []topStockItemResponse `json:"total"`
	ResponseInfo  responseInfoResp       `json:"response_info"`
	DisplayOption displayOptionResp      `json:"display_option"`
}

type topStockItemResponse struct {
	Rank         int          `json:"rank"`
	Code         string       `json:"code"`
	IconURL      string       `json:"icon_url"`
	Value        rawFormatted `json:"value"`
	Lot          rawFormatted `json:"lot"`
	Average      rawFormatted `json:"average"`
	ForeignValue rawFormatted `json:"foreign_value"`
	Frequency    rawFormatted `json:"frequency"`
}

type rawFormatted struct {
	Raw       string `json:"raw"`
	Formatted string `json:"formatted"`
}

type responseInfoResp struct {
	Page           int    `json:"page"`
	Limit          int    `json:"limit"`
	MaxDayDuration int    `json:"max_day_duration"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
	ValueType      string `json:"value_type"`
}

type displayOptionResp struct {
	BannerMessage      string           `json:"banner_message"`
	ForeignValueColumn bool             `json:"foreign_value_column"`
	EnabledValueType   enabledValueResp `json:"enabled_value_type"`
}

type enabledValueResp struct {
	Gross bool `json:"gross"`
	Net   bool `json:"net"`
	Total bool `json:"total"`
}

func (h *TopStockHandler) TopStock(w http.ResponseWriter, r *http.Request) {
	req := topStockRequest{
		Start:        r.URL.Query().Get("start"),
		End:          r.URL.Query().Get("end"),
		InvestorType: r.URL.Query().Get("investor_type"),
		MarketType:   r.URL.Query().Get("market_type"),
		ValueType:    r.URL.Query().Get("value_type"),
	}
	if req.InvestorType == "" {
		req.InvestorType = defaultInvestorType
	}
	if req.MarketType == "" {
		req.MarketType = defaultMarketType
	}
	if req.ValueType == "" {
		req.ValueType = defaultValueType
	}
	if v := r.URL.Query().Get("page"); v != "" {
		req.Page, _ = strconv.Atoi(v)
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate top stock params")
		return
	}

	data, err := h.uc.GetTopStock(r.Context(), req.Start, req.End, req.InvestorType, req.MarketType, req.ValueType, req.Page)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get top stock data")
		return
	}
	response.OK(w, toResponse(data))
}

func toResponse(d *domain.TopStockData) topStockResponse {
	ri := d.ResponseInfo
	di := d.DisplayOption
	return topStockResponse{
		TopBuy:  toItemResponses(d.TopBuy),
		TopSell: toItemResponses(d.TopSell),
		Total:   toItemResponses(d.Total),
		ResponseInfo: responseInfoResp{
			Page:           ri.Page,
			Limit:          ri.Limit,
			MaxDayDuration: ri.MaxDayDuration,
			StartDate:      ri.StartDate,
			EndDate:        ri.EndDate,
			ValueType:      ri.ValueType,
		},
		DisplayOption: displayOptionResp{
			BannerMessage:      di.BannerMessage,
			ForeignValueColumn: di.ForeignValueColumn,
			EnabledValueType: enabledValueResp{
				Gross: di.EnabledValueType.Gross,
				Net:   di.EnabledValueType.Net,
				Total: di.EnabledValueType.Total,
			},
		},
	}
}

func toItemResponses(items []domain.TopStockItem) []topStockItemResponse {
	out := make([]topStockItemResponse, 0, len(items))
	for _, i := range items {
		out = append(out, topStockItemResponse{
			Rank:         i.Rank,
			Code:         i.Code,
			IconURL:      i.IconURL,
			Value:        rawFormatted{Raw: i.Value.Raw, Formatted: i.Value.Formatted},
			Lot:          rawFormatted{Raw: i.Lot.Raw, Formatted: i.Lot.Formatted},
			Average:      rawFormatted{Raw: i.Average.Raw, Formatted: i.Average.Formatted},
			ForeignValue: rawFormatted{Raw: i.ForeignValue.Raw, Formatted: i.ForeignValue.Formatted},
			Frequency:    rawFormatted{Raw: i.Frequency.Raw, Formatted: i.Frequency.Formatted},
		})
	}
	return out
}
