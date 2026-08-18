package historicalsummary

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

const (
	defaultPeriod = "HS_PERIOD_DAILY"
	defaultLimit  = 50
	defaultPage   = 1
)

type HistoricalSummaryHandler struct {
	uc usecase.HistoricalSummaryUsecase
	v  validator.Validator
}

func NewHistoricalSummaryHandler(uc usecase.HistoricalSummaryUsecase, v validator.Validator) *HistoricalSummaryHandler {
	return &HistoricalSummaryHandler{uc: uc, v: v}
}

type historicalSummaryRequest struct {
	Symbol    string `json:"symbol" validate:"required"`
	Period    string `json:"period" validate:"omitempty,oneof=HS_PERIOD_DAILY HS_PERIOD_WEEKLY HS_PERIOD_MONTHLY"`
	StartDate string `json:"start_date" validate:"omitempty,datetime=2006-01-02"`
	EndDate   string `json:"end_date" validate:"omitempty,datetime=2006-01-02"`
	Limit     int    `json:"limit" validate:"min=1"`
	Page      int    `json:"page" validate:"min=1"`
}

type historicalSummaryResponse struct {
	Result   []historicalSummaryItemResp   `json:"result"`
	Paginate historicalSummaryPaginateResp `json:"paginate"`
}

type historicalSummaryItemResp struct {
	Date             string  `json:"date"`
	Close            float64 `json:"close"`
	Change           float64 `json:"change"`
	Value            int64   `json:"value"`
	Volume           int64   `json:"volume"`
	Frequency        int64   `json:"frequency"`
	ForeignBuy       int64   `json:"foreign_buy"`
	ForeignSell      int64   `json:"foreign_sell"`
	NetForeign       int64   `json:"net_foreign"`
	Open             float64 `json:"open"`
	High             float64 `json:"high"`
	Low              float64 `json:"low"`
	Average          float64 `json:"average"`
	ChangePercentage float64 `json:"change_percentage"`
}

type historicalSummaryPaginateResp struct {
	NextPage string `json:"next_page"`
}

func parseIntQuery(s string, def int) (int, bool) {
	if s == "" {
		return def, true
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

func (h *HistoricalSummaryHandler) HistoricalSummary(w http.ResponseWriter, r *http.Request) {
	limit, ok := parseIntQuery(r.URL.Query().Get("limit"), defaultLimit)
	if !ok {
		response.ValidationError(w, "validation failed", map[string]string{"limit": "must be a valid integer"})
		return
	}
	page, ok := parseIntQuery(r.URL.Query().Get("page"), defaultPage)
	if !ok {
		response.ValidationError(w, "validation failed", map[string]string{"page": "must be a valid integer"})
		return
	}

	req := historicalSummaryRequest{
		Symbol:    chi.URLParam(r, "symbol"),
		Period:    r.URL.Query().Get("period"),
		StartDate: r.URL.Query().Get("start_date"),
		EndDate:   r.URL.Query().Get("end_date"),
		Limit:     limit,
		Page:      page,
	}
	if req.Period == "" {
		req.Period = defaultPeriod
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate historical summary params")
		return
	}

	data, err := h.uc.GetHistoricalSummary(r.Context(), req.Symbol, req.Period, req.StartDate, req.EndDate, req.Limit, req.Page)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get historical summary")
		return
	}
	response.OK(w, toResponse(data))
}

func toResponse(d *domain.HistoricalSummaryData) historicalSummaryResponse {
	out := historicalSummaryResponse{
		Result:   make([]historicalSummaryItemResp, 0, len(d.Result)),
		Paginate: historicalSummaryPaginateResp{NextPage: d.Paginate.NextPage},
	}
	for _, it := range d.Result {
		out.Result = append(out.Result, historicalSummaryItemResp{
			Date:             it.Date,
			Close:            it.Close,
			Change:           it.Change,
			Value:            it.Value,
			Volume:           it.Volume,
			Frequency:        it.Frequency,
			ForeignBuy:       it.ForeignBuy,
			ForeignSell:      it.ForeignSell,
			NetForeign:       it.NetForeign,
			Open:             it.Open,
			High:             it.High,
			Low:              it.Low,
			Average:          it.Average,
			ChangePercentage: it.ChangePercentage,
		})
	}
	return out
}
