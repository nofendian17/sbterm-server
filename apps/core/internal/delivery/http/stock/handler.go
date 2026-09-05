// Package http provides HTTP handlers for the core service API.

package stock

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

// StockHandler serves the stock catalog. Read methods are user-facing
// (stocks:read); admin methods are gated by the router (stocks:write / sync).
type StockHandler struct {
	uc usecase.StockUsecase
	v  validator.Validator
}

func NewStockHandler(uc usecase.StockUsecase, v validator.Validator) *StockHandler {
	return &StockHandler{uc: uc, v: v}
}

type sectorResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type stockResponse struct {
	Symbol    string          `json:"symbol"`
	Name      string          `json:"name"`
	Sector    *sectorResponse `json:"sector,omitempty"`
	Exchange  *string         `json:"exchange,omitempty"`
	IconURL   *string         `json:"icon_url,omitempty"`
	IsActive  bool            `json:"is_active"`
	SyncedAt  *string         `json:"synced_at,omitempty"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

func toStockResponse(s domain.Stock) stockResponse {
	out := stockResponse{
		Symbol:    s.Symbol,
		Name:      s.Name,
		Exchange:  s.Exchange,
		IconURL:   s.IconURL,
		IsActive:  s.IsActive,
		CreatedAt: formatTime(s.CreatedAt),
		UpdatedAt: formatTime(s.UpdatedAt),
	}
	if s.Sector != nil {
		out.Sector = &sectorResponse{ID: s.Sector.ID, Name: s.Sector.Name}
	}
	if s.SyncedAt != nil {
		v := formatTime(*s.SyncedAt)
		out.SyncedAt = &v
	}
	return out
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// --- User-facing reads -----------------------------------------------------

func (h *StockHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	f := domain.StockFilter{
		Query:  q.Get("q"),
		Sector: q.Get("sector"),
		Page:   page,
		Limit:  limit,
	}
	if raw, ok := q["active"]; ok && len(raw) > 0 {
		if v, err := strconv.ParseBool(raw[0]); err == nil {
			f.IsActive = &v
		}
	}

	stocks, total, err := h.uc.List(r.Context(), f)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}

	resp := make([]stockResponse, len(stocks))
	for i, s := range stocks {
		resp[i] = toStockResponse(s)
	}
	pg, lim := domain.NormalizePagination(page, limit)
	totalPages := 0
	if total > 0 {
		totalPages = (total + lim - 1) / lim
	}
	response.Paginated(w, resp, &response.MetaBody{
		Page:       pg,
		Limit:      lim,
		TotalItems: total,
		TotalPages: totalPages,
	})
}

func (h *StockHandler) GetBySymbol(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "symbol is required")
		return
	}
	s, err := h.uc.GetBySymbol(r.Context(), symbol)
	if err != nil {
		if errors.Is(err, domain.ErrStockNotFound) {
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "stock not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	response.OK(w, toStockResponse(s))
}

// --- Admin: write / sync ---------------------------------------------------

type createStockRequest struct {
	Symbol   string  `json:"symbol" validate:"required"`
	Name     string  `json:"name" validate:"required"`
	Sector   *string `json:"sector,omitempty"`
	Exchange *string `json:"exchange,omitempty"`
	IconURL  *string `json:"icon_url,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

func (h *StockHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}

	s, err := h.uc.Create(r.Context(), domain.StockCreateInput{
		Symbol:   req.Symbol,
		Name:     req.Name,
		Sector:   req.Sector,
		Exchange: req.Exchange,
		IconURL:  req.IconURL,
		IsActive: req.IsActive,
	})
	if err != nil {
		mapStockError(w, err)
		return
	}
	response.Created(w, toStockResponse(s))
}

type updateStockRequest struct {
	Name     *string `json:"name,omitempty"`
	Sector   *string `json:"sector,omitempty"`
	Exchange *string `json:"exchange,omitempty"`
	IconURL  *string `json:"icon_url,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

func (h *StockHandler) Update(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "symbol is required")
		return
	}
	var req updateStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	if err := h.uc.Update(r.Context(), symbol, domain.StockUpdateInput{
		Name:     req.Name,
		Sector:   req.Sector,
		Exchange: req.Exchange,
		IconURL:  req.IconURL,
		IsActive: req.IsActive,
	}); err != nil {
		mapStockError(w, err)
		return
	}
	response.Message(w, http.StatusOK, "stock updated")
}

func (h *StockHandler) Delete(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "symbol is required")
		return
	}
	if err := h.uc.SoftDelete(r.Context(), symbol); err != nil {
		mapStockError(w, err)
		return
	}
	response.NoContent(w)
}

func (h *StockHandler) Sync(w http.ResponseWriter, r *http.Request) {
	res, err := h.uc.SyncAll(r.Context())
	if err != nil {
		response.Error(w, http.StatusBadGateway, response.CodeUpstreamError, "stock sync failed")
		return
	}
	response.OK(w, res)
}

// mapStockError maps domain errors to HTTP responses for admin stock ops.
func mapStockError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "validation failed")
	case errors.Is(err, domain.ErrSectorNotFound):
		response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "unknown sector")
	case errors.Is(err, domain.ErrStockNotFound):
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "stock not found")
	case errors.Is(err, domain.ErrStockSymbolTaken):
		response.Error(w, http.StatusConflict, response.CodeConflict, "stock symbol already exists")
	case errors.Is(err, domain.ErrStockHasWatchlists):
		response.Error(w, http.StatusConflict, response.CodeConflict, "stock has active watchlists")
	case errors.Is(err, domain.ErrStockSyncFailed):
		response.Error(w, http.StatusBadGateway, response.CodeUpstreamError, "stock sync failed")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
	}
}
