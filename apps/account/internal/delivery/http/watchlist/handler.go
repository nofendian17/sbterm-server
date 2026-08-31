package watchlist

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/account/internal/delivery/http/middleware"
	"github.com/nofendian17/sbterm/apps/account/internal/domain"
	"github.com/nofendian17/sbterm/apps/account/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
)

type WatchlistHandler struct {
	uc usecase.WatchlistUsecase
}

func NewWatchlistHandler(uc usecase.WatchlistUsecase) *WatchlistHandler {
	return &WatchlistHandler{uc: uc}
}

type addWatchlistRequest struct {
	Symbol string `json:"symbol"`
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

	err := h.uc.Add(r.Context(), userID, domain.AddWatchlistInput{
		Symbol: req.Symbol,
		Label:  req.Label,
	})
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateWatchlist) {
			response.Error(w, http.StatusConflict, response.CodeConflict, "symbol already in watchlist")
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
