package foreigndomestic

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/response"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

// Default values applied when the corresponding query param is omitted.
const (
	defaultMarketType = "MARKET_TYPE_ALL"
	defaultPeriod     = "TB_PERIOD_LAST_1_DAY"
)

type ForeignDomesticHandler struct {
	uc usecase.ForeignDomesticUsecase
	v  validator.Validator
}

func NewForeignDomesticHandler(uc usecase.ForeignDomesticUsecase, v validator.Validator) *ForeignDomesticHandler {
	return &ForeignDomesticHandler{uc: uc, v: v}
}

type foreignDomesticRequest struct {
	Symbol     string `json:"symbol" validate:"required"`
	MarketType string `json:"market_type" validate:"omitempty,oneof=MARKET_TYPE_ALL"`
	Period     string `json:"period" validate:"omitempty,oneof=TB_PERIOD_LAST_1_DAY TB_PERIOD_LAST_7_DAYS TB_PERIOD_LAST_1_MONTH TB_PERIOD_YEAR_TO_DATE TB_PERIOD_LAST_1_YEAR"`
	From       string `json:"from" validate:"omitempty,datetime=2006-01-02"`
	To         string `json:"to" validate:"omitempty,datetime=2006-01-02"`
}

type foreignDomesticResponse struct {
	HistoricalPrice []foreignDomesticPricePointResp `json:"historical_price"`
	HistoricalNet   []foreignDomesticNetPointResp   `json:"historical_net"`
	LastUpdated     string                          `json:"last_updated"`
	From            string                          `json:"from"`
	To              string                          `json:"to"`
}

type foreignDomesticPricePointResp struct {
	Date          string       `json:"date"`
	DatetimeLabel string       `json:"datetime_label"`
	Open          rawFormatted `json:"open"`
	High          rawFormatted `json:"high"`
	Low           rawFormatted `json:"low"`
	Close         rawFormatted `json:"close"`
}

type foreignDomesticNetPointResp struct {
	Date                    string    `json:"date"`
	DatetimeLabel           string    `json:"datetime_label"`
	DatetimeLabelTable      string    `json:"datetime_label_table"`
	NetForeign              valueResp `json:"net_foreign"`
	ForeignBuy              valueResp `json:"foreign_buy"`
	ForeignSell             valueResp `json:"foreign_sell"`
	ForeignFlow             valueResp `json:"foreign_flow"`
	NetLot                  valueResp `json:"net_lot"`
	NetFrequency            valueResp `json:"net_frequency"`
	AveragePrice            valueResp `json:"average_price"`
	PercentageForeignValue  valueResp `json:"percentage_foreign_value"`
	PercentageDomesticValue valueResp `json:"percentage_domestic_value"`
}

type rawFormatted struct {
	Raw       string `json:"raw"`
	Formatted string `json:"formatted"`
}

type valueResp struct {
	Raw       json.Number `json:"raw"`
	Formatted string      `json:"formatted"`
}

// foreignDomesticRangeRequirements returns per-field validation messages for
// the from/to pair, or nil when well-formed. from/to must either both be
// provided or both omitted (a single bound is silently ignored upstream, which
// is a footgun); a reversed range is rejected because the upstream 400s on it.
// Dates are validated as YYYY-MM-DD, so string comparison equals date ordering.
func foreignDomesticRangeRequirements(req foreignDomesticRequest) map[string]string {
	if (req.From == "") != (req.To == "") {
		if req.From == "" {
			return map[string]string{"from": "from and to must both be provided or both omitted"}
		}
		return map[string]string{"to": "from and to must both be provided or both omitted"}
	}
	if req.From != "" && req.From > req.To {
		return map[string]string{"from": "must be earlier than or equal to to"}
	}
	return nil
}

func (h *ForeignDomesticHandler) ForeignDomesticHistorical(w http.ResponseWriter, r *http.Request) {
	req := foreignDomesticRequest{
		Symbol:     r.URL.Query().Get("symbol"),
		MarketType: r.URL.Query().Get("market_type"),
		Period:     r.URL.Query().Get("period"),
		From:       r.URL.Query().Get("from"),
		To:         r.URL.Query().Get("to"),
	}
	if req.MarketType == "" {
		req.MarketType = defaultMarketType
	}
	if req.Period == "" {
		req.Period = defaultPeriod
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate foreign domestic params")
		return
	}
	if fields := foreignDomesticRangeRequirements(req); len(fields) > 0 {
		response.ValidationError(w, "validation failed", fields)
		return
	}

	data, err := h.uc.GetForeignDomesticHistorical(r.Context(), req.Symbol, req.MarketType, req.Period, req.From, req.To)
	if err != nil {
		var upErr *domain.UpstreamError
		if errors.As(err, &upErr) && upErr.Status == http.StatusBadRequest {
			response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "no foreign domestic data for the requested parameters")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get foreign domestic historical")
		return
	}
	response.OK(w, toResponse(data))
}

func toResponse(d *domain.ForeignDomesticData) foreignDomesticResponse {
	out := foreignDomesticResponse{
		LastUpdated:     d.LastUpdated,
		From:            d.From,
		To:              d.To,
		HistoricalPrice: make([]foreignDomesticPricePointResp, 0, len(d.HistoricalPrice)),
		HistoricalNet:   make([]foreignDomesticNetPointResp, 0, len(d.HistoricalNet)),
	}
	for _, p := range d.HistoricalPrice {
		out.HistoricalPrice = append(out.HistoricalPrice, foreignDomesticPricePointResp{
			Date:          p.Date,
			DatetimeLabel: p.DatetimeLabel,
			Open:          toRawFormatted(p.Open),
			High:          toRawFormatted(p.High),
			Low:           toRawFormatted(p.Low),
			Close:         toRawFormatted(p.Close),
		})
	}
	for _, n := range d.HistoricalNet {
		out.HistoricalNet = append(out.HistoricalNet, foreignDomesticNetPointResp{
			Date:                    n.Date,
			DatetimeLabel:           n.DatetimeLabel,
			DatetimeLabelTable:      n.DatetimeLabelTable,
			NetForeign:              toValue(n.NetForeign),
			ForeignBuy:              toValue(n.ForeignBuy),
			ForeignSell:             toValue(n.ForeignSell),
			ForeignFlow:             toValue(n.ForeignFlow),
			NetLot:                  toValue(n.NetLot),
			NetFrequency:            toValue(n.NetFrequency),
			AveragePrice:            toValue(n.AveragePrice),
			PercentageForeignValue:  toValue(n.PercentageForeignValue),
			PercentageDomesticValue: toValue(n.PercentageDomesticValue),
		})
	}
	return out
}

func toRawFormatted(v domain.RawFormatted) rawFormatted {
	return rawFormatted{Raw: v.Raw, Formatted: v.Formatted}
}

func toValue(v domain.ForeignDomesticValue) valueResp {
	return valueResp{Raw: v.Raw, Formatted: v.Formatted}
}
