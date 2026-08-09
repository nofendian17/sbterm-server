package findata

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/response"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

type FindataFinancialHandler struct {
	uc usecase.FindataFinancialUsecase
	v  validator.Validator
}

func NewFindataFinancialHandler(uc usecase.FindataFinancialUsecase, v validator.Validator) *FindataFinancialHandler {
	return &FindataFinancialHandler{uc: uc, v: v}
}

// reportType: 1=Income Statement, 2=Balance Sheet, 3=Cash Flow.
// statementType: 1=Quarterly 2=Annual 3=TTM 4=Interim YTD 5..8=Q1..Q4
// 9=QoQ Growth 10=Quarter YoY 11=YTD YoY 12=Annual YoY 13=3Y CAGR.
type findataFinancialRequest struct {
	Symbol        string `json:"symbol" validate:"required"`
	DataType      int    `json:"data_type" validate:"omitempty"`
	IsPercentage  int    `json:"is_percentage" validate:"omitempty"` // 0 | 1
	Page          int    `json:"page" validate:"required,min=1"`
	ReportType    int    `json:"report_type" validate:"required,oneof=1 2 3"`
	StatementType int    `json:"statement_type" validate:"required,oneof=1 2 3 4 5 6 7 8 9 10 11 12 13"`
}

type findataFinancialResponse struct {
	Currency        []string                  `json:"currency"`
	DefaultCurrency string                    `json:"default_currency"`
	RoundingValue   []int                     `json:"rounding_value"`
	DataTables      findataDataTablesResponse `json:"data_tables"`
}

type findataDataTablesResponse struct {
	Periods      []string                 `json:"periods"`
	Accounts     []findataAccountResponse `json:"accounts"`
	MaxShowLevel int                      `json:"max_show_level"`
}

type findataAccountResponse struct {
	ID                int64                    `json:"id"`
	Level             int                      `json:"level"`
	Name              string                   `json:"name"`
	Values            []string                 `json:"values"`
	Accounts          []findataAccountResponse `json:"accounts"`
	IsTotalExist      bool                     `json:"is_total_exist"`
	IsDefaultExpanded bool                     `json:"is_default_expanded"`
	MaxShowLevel      int                      `json:"max_show_level"`
}

func (h *FindataFinancialHandler) Financial(w http.ResponseWriter, r *http.Request) {
	req := findataFinancialRequest{Symbol: chi.URLParam(r, "symbol")}
	parse := func(name string) int {
		if v := r.URL.Query().Get(name); v != "" {
			n, _ := strconv.Atoi(v)
			return n
		}
		return 0
	}
	req.DataType = parse("data_type")
	req.IsPercentage = parse("is_percentage")
	req.Page = parse("page")
	req.ReportType = parse("report_type")
	req.StatementType = parse("statement_type")
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate financial params")
		return
	}

	data, err := h.uc.GetFindataFinancial(r.Context(), req.Symbol, req.DataType, req.IsPercentage, req.Page, req.ReportType, req.StatementType)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get financial report")
		return
	}
	response.OK(w, toResponse(data))
}

func toResponse(d *domain.FindataFinancial) findataFinancialResponse {
	out := findataFinancialResponse{
		Currency:        d.Currency,
		DefaultCurrency: d.DefaultCurrency,
		RoundingValue:   d.RoundingValue,
		DataTables: findataDataTablesResponse{
			Periods:      d.DataTables.Periods,
			MaxShowLevel: d.DataTables.MaxShowLevel,
			Accounts:     toAccountResponses(d.DataTables.Accounts),
		},
	}
	return out
}

func toAccountResponses(in []domain.FindataAccount) []findataAccountResponse {
	out := make([]findataAccountResponse, 0, len(in))
	for _, a := range in {
		out = append(out, findataAccountResponse{
			ID:                a.ID,
			Level:             a.Level,
			Name:              a.Name,
			Values:            a.Values,
			Accounts:          toAccountResponses(a.Accounts),
			IsTotalExist:      a.IsTotalExist,
			IsDefaultExpanded: a.IsDefaultExpanded,
			MaxShowLevel:      a.MaxShowLevel,
		})
	}
	return out
}
