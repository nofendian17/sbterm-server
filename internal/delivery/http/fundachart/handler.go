package fundachart

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/response"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

type FundaChartHandler struct {
	uc usecase.FundaChartUsecase
	v  validator.Validator
}

func NewFundaChartHandler(uc usecase.FundaChartUsecase, v validator.Validator) *FundaChartHandler {
	return &FundaChartHandler{uc: uc, v: v}
}

type fundaChartRequest struct {
	Symbol    string `json:"symbol" validate:"required"`
	Item      string `json:"item" validate:"required"`
	Timeframe string `json:"timeframe" validate:"omitempty,oneof=1y 3y 5y 10y"`
}

type fundaChartResponse struct {
	CompanyID   int64           `json:"company_id"`
	CompanyName string          `json:"company_name"`
	Ratios      []ratioResponse `json:"ratios"`
}

type ratioResponse struct {
	DecimalPoint int                  `json:"decimal_point"`
	GroupData    bool                 `json:"group_data"`
	ItemID       int64                `json:"item_id"`
	ItemName     string               `json:"item_name"`
	ItemType     int                  `json:"item_type"`
	Suffix       string               `json:"suffix"`
	XAxisID      int                  `json:"xaxis_id"`
	YAxisID      int                  `json:"yaxis_id"`
	ChartData    []chartPointResponse `json:"chart_data"`
}

type chartPointResponse struct {
	Date         int64   `json:"date"`
	FormatedDate string  `json:"formated_date"`
	Value        float64 `json:"value"`
	RatioValue   float64 `json:"ratio_value"`
}

func (h *FundaChartHandler) FundaChart(w http.ResponseWriter, r *http.Request) {
	req := fundaChartRequest{Symbol: chi.URLParam(r, "symbol"), Item: r.URL.Query().Get("item")}
	req.Timeframe = r.URL.Query().Get("timeframe")
	if req.Timeframe == "" {
		req.Timeframe = "10y"
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate funda chart params")
		return
	}

	data, err := h.uc.GetFundaChart(r.Context(), req.Symbol, req.Item, req.Timeframe)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get funda chart")
		return
	}
	response.OK(w, toResponses(data))
}

func toResponses(in []domain.FundaChartCompany) []fundaChartResponse {
	res := make([]fundaChartResponse, 0, len(in))
	for _, c := range in {
		out := fundaChartResponse{CompanyID: c.CompanyID, CompanyName: c.CompanyName, Ratios: make([]ratioResponse, 0, len(c.Ratios))}
		for _, rt := range c.Ratios {
			rr := ratioResponse{
				DecimalPoint: rt.DecimalPoint,
				GroupData:    rt.GroupData,
				ItemID:       rt.ItemID,
				ItemName:     rt.ItemName,
				ItemType:     rt.ItemType,
				Suffix:       rt.Suffix,
				XAxisID:      rt.XAxisID,
				YAxisID:      rt.YAxisID,
				ChartData:    make([]chartPointResponse, 0, len(rt.ChartData)),
			}
			for _, p := range rt.ChartData {
				rr.ChartData = append(rr.ChartData, chartPointResponse{
					Date:         p.Date,
					FormatedDate: p.FormatedDate,
					Value:        p.Value,
					RatioValue:   p.RatioValue,
				})
			}
			out.Ratios = append(out.Ratios, rr)
		}
		res = append(res, out)
	}
	return res
}
