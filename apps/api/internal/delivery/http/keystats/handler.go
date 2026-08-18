package keystats

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

type KeystatsHandler struct {
	uc usecase.KeystatsUsecase
	v  validator.Validator
}

func NewKeystatsHandler(uc usecase.KeystatsUsecase, v validator.Validator) *KeystatsHandler {
	return &KeystatsHandler{uc: uc, v: v}
}

type keystatsRequest struct {
	Symbol    string `json:"symbol" validate:"required"`
	YearLimit int    `json:"year_limit" validate:"omitempty"`
}

type keystatsResponse struct {
	ClosureFinItemsResults  []finGroupResponse    `json:"closure_fin_items_results"`
	FinancialYearParent     yearParentResponse    `json:"financial_year_parent"`
	Stats                   statsResponse         `json:"stats"`
	Info                    string                `json:"info"`
	DividendGroup           dividendGroupResponse `json:"dividend_group"`
	FinancialReportCurrency []string              `json:"financial_report_currency"`
}

type finGroupResponse struct {
	KeystatsName   string         `json:"keystats_name"`
	FinNameResults []itemResponse `json:"fin_name_results"`
}

type itemResponse struct {
	Fitem          fitemResponse `json:"fitem"`
	IsNewUpdate    bool          `json:"is_new_update"`
	HiddenGraphIco bool          `json:"hidden_graph_ico"`
}

type fitemResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type yearParentResponse struct {
	FinancialYearGroups    []yearGroupResponse `json:"financial_year_groups"`
	FinancialYearGroupsUSD []yearGroupResponse `json:"financial_year_groups_usd"`
}

type yearGroupResponse struct {
	FinancialYearValues []yearResponse `json:"financial_year_values"`
}

type yearResponse struct {
	Year            string           `json:"year"`
	PeriodValues    []periodResponse `json:"period_values"`
	AnnualisedValue string           `json:"annualised_value"`
	TTMValue        string           `json:"ttm_value"`
	IsNewUpdate     bool             `json:"is_new_update"`
	Dividend        string           `json:"dividend"`
	PayoutRatio     string           `json:"payout_ratio"`
	DividendYield   string           `json:"dividend_yield"`
}

type periodResponse struct {
	Period       string `json:"period"`
	Year         string `json:"year"`
	QuarterValue string `json:"quarter_value"`
	IsNewUpdate  bool   `json:"is_new_update"`
}

type statsResponse struct {
	CurrentShareOutstanding string `json:"current_share_outstanding"`
	MarketCap               string `json:"market_cap"`
	EnterpriseValue         string `json:"enterprise_value"`
	FreeFloat               string `json:"free_float"`
}

type dividendGroupResponse struct {
	FitemID            []string               `json:"fitem_id"`
	DividendYearValues []dividendYearResponse `json:"dividend_year_values"`
}

type dividendYearResponse struct {
	Period      int    `json:"period"`
	Dividend    string `json:"dividend"`
	ExDate      string `json:"ex_date"`
	PaymentDate string `json:"payment_date"`
}

func (h *KeystatsHandler) Keystats(w http.ResponseWriter, r *http.Request) {
	req := keystatsRequest{Symbol: chi.URLParam(r, "symbol")}
	if v := r.URL.Query().Get("year_limit"); v != "" {
		req.YearLimit, _ = strconv.Atoi(v)
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate keystats params")
		return
	}

	keystats, err := h.uc.GetKeystats(r.Context(), req.Symbol, req.YearLimit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get keystats")
		return
	}
	response.OK(w, toResponse(keystats))
}

func toResponse(k *domain.Keystats) keystatsResponse {
	out := keystatsResponse{
		Info:                    k.Info,
		Stats:                   toStatsResponse(k.Stats),
		FinancialYearParent:     toYearParentResponse(k.FinancialYearParent),
		DividendGroup:           toDividendGroupResponse(k.DividendGroup),
		FinancialReportCurrency: k.FinancialReportCurrency,
		ClosureFinItemsResults:  make([]finGroupResponse, 0, len(k.ClosureFinItemsResults)),
	}
	for _, g := range k.ClosureFinItemsResults {
		group := finGroupResponse{
			KeystatsName:   g.KeystatsName,
			FinNameResults: make([]itemResponse, 0, len(g.FinNameResults)),
		}
		for _, it := range g.FinNameResults {
			group.FinNameResults = append(group.FinNameResults, itemResponse{
				Fitem:          fitemResponse{ID: it.Fitem.ID, Name: it.Fitem.Name, Value: it.Fitem.Value},
				IsNewUpdate:    it.IsNewUpdate,
				HiddenGraphIco: it.HiddenGraphIco,
			})
		}
		out.ClosureFinItemsResults = append(out.ClosureFinItemsResults, group)
	}
	return out
}

func toStatsResponse(s domain.KeystatsStats) statsResponse {
	return statsResponse{
		CurrentShareOutstanding: s.CurrentShareOutstanding,
		MarketCap:               s.MarketCap,
		EnterpriseValue:         s.EnterpriseValue,
		FreeFloat:               s.FreeFloat,
	}
}

func toYearParentResponse(p domain.KeystatsYearParent) yearParentResponse {
	return yearParentResponse{
		FinancialYearGroups:    toYearGroupsResponse(p.FinancialYearGroups),
		FinancialYearGroupsUSD: toYearGroupsResponse(p.FinancialYearGroupsUSD),
	}
}

func toYearGroupsResponse(in []domain.KeystatsYearGroup) []yearGroupResponse {
	out := make([]yearGroupResponse, 0, len(in))
	for _, g := range in {
		group := yearGroupResponse{
			FinancialYearValues: make([]yearResponse, 0, len(g.FinancialYearValues)),
		}
		for _, y := range g.FinancialYearValues {
			year := yearResponse{
				Year:            y.Year,
				AnnualisedValue: y.AnnualisedValue,
				TTMValue:        y.TTMValue,
				IsNewUpdate:     y.IsNewUpdate,
				Dividend:        y.Dividend,
				PayoutRatio:     y.PayoutRatio,
				DividendYield:   y.DividendYield,
				PeriodValues:    make([]periodResponse, 0, len(y.PeriodValues)),
			}
			for _, p := range y.PeriodValues {
				year.PeriodValues = append(year.PeriodValues, periodResponse{
					Period:       p.Period,
					Year:         p.Year,
					QuarterValue: p.QuarterValue,
					IsNewUpdate:  p.IsNewUpdate,
				})
			}
			group.FinancialYearValues = append(group.FinancialYearValues, year)
		}
		out = append(out, group)
	}
	return out
}

func toDividendGroupResponse(d domain.KeystatsDividendGroup) dividendGroupResponse {
	out := dividendGroupResponse{
		FitemID:            d.FitemID,
		DividendYearValues: make([]dividendYearResponse, 0, len(d.DividendYearValues)),
	}
	for _, y := range d.DividendYearValues {
		out.DividendYearValues = append(out.DividendYearValues, dividendYearResponse{
			Period:      y.Period,
			Dividend:    y.Dividend,
			ExDate:      y.ExDate,
			PaymentDate: y.PaymentDate,
		})
	}
	return out
}
