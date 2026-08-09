package fundachart

import (
	"net/http"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/response"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

type FundaChartMetricsHandler struct {
	uc usecase.FundaChartMetricsUsecase
	v  validator.Validator
}

func NewFundaChartMetricsHandler(uc usecase.FundaChartMetricsUsecase, v validator.Validator) *FundaChartMetricsHandler {
	return &FundaChartMetricsHandler{uc: uc, v: v}
}

type metricsRequest struct {
	MetricName string `json:"metric_name" validate:"required"`
}

type metricResponse struct {
	FitemID       int64            `json:"fitem_id"`
	FitemName     string           `json:"fitem_name"`
	ShowChartIcon int              `json:"show_chart_icon"`
	Child         []metricResponse `json:"child"`
}

func (h *FundaChartMetricsHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	req := metricsRequest{MetricName: r.URL.Query().Get("metric_name")}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate metrics params")
		return
	}

	metrics, err := h.uc.GetFundaChartMetrics(r.Context(), req.MetricName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get funda chart metrics")
		return
	}
	response.OK(w, toMetricResponses(metrics))
}

func toMetricResponses(in []domain.FundaChartMetric) []metricResponse {
	res := make([]metricResponse, 0, len(in))
	for _, m := range in {
		res = append(res, toMetricResponse(m))
	}
	return res
}

func toMetricResponse(m domain.FundaChartMetric) metricResponse {
	out := metricResponse{
		FitemID:       m.FitemID,
		FitemName:     m.FitemName,
		ShowChartIcon: m.ShowChartIcon,
		Child:         make([]metricResponse, 0, len(m.Child)),
	}
	for _, c := range m.Child {
		out.Child = append(out.Child, toMetricResponse(c))
	}
	return out
}
