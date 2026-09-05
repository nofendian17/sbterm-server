// Package http provides HTTP handlers for the core service API.

package watchlist

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/middleware"
	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

type WatchlistHandler struct {
	uc usecase.WatchlistUsecase
	v  validator.Validator
}

func NewWatchlistHandler(uc usecase.WatchlistUsecase, v validator.Validator) *WatchlistHandler {
	return &WatchlistHandler{uc: uc, v: v}
}

type addWatchlistRequest struct {
	Symbol string `json:"symbol" validate:"required"`
	Label  string `json:"label,omitempty"`
}

func (h *WatchlistHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "unauthorized")
		return
	}

	items, err := h.uc.List(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}

	response.OK(w, items)
}

// ListByAdmin returns the watchlists for the user identified by the {id}
// path parameter. It is used by the admin route to view another user's
// watchlists instead of the requesting admin's own data.
func (h *WatchlistHandler) ListByAdmin(w http.ResponseWriter, r *http.Request) {
	targetUserID := chi.URLParam(r, "id")
	if targetUserID == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "user id is required")
		return
	}

	items, err := h.uc.List(r.Context(), targetUserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}

	response.OK(w, items)
}

func (h *WatchlistHandler) Add(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "unauthorized")
		return
	}

	var req addWatchlistRequest
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

	err := h.uc.Add(r.Context(), userID, domain.AddWatchlistInput{
		Symbol: req.Symbol,
		Label:  req.Label,
	})
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateWatchlist) {
			response.Error(w, http.StatusConflict, response.CodeConflict, "symbol already in watchlist")
			return
		}
		if errors.Is(err, domain.ErrStockNotFound) {
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "stock not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}

	response.Created(w, map[string]string{"symbol": req.Symbol})
}

func (h *WatchlistHandler) Remove(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "unauthorized")
		return
	}

	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "symbol is required")
		return
	}

	if err := h.uc.Remove(r.Context(), userID, symbol); err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}

	response.NoContent(w)
}
