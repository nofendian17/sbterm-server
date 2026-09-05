// Package sector provides HTTP handlers for the sectors master table.

package sector

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

// SectorHandler serves sectors. Reads are user-facing (stocks:read);
// writes are admin-gated by the router (stocks:write).
type SectorHandler struct {
	uc usecase.SectorUsecase
	v  validator.Validator
}

func NewSectorHandler(uc usecase.SectorUsecase, v validator.Validator) *SectorHandler {
	return &SectorHandler{uc: uc, v: v}
}

type sectorResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *SectorHandler) List(w http.ResponseWriter, r *http.Request) {
	sectors, err := h.uc.List(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	resp := make([]sectorResponse, len(sectors))
	for i, s := range sectors {
		resp[i] = sectorResponse{ID: s.ID, Name: s.Name}
	}
	response.OK(w, resp)
}

func (h *SectorHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "id is required")
		return
	}
	s, err := h.uc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrSectorNotFound) {
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "sector not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	response.OK(w, sectorResponse{ID: s.ID, Name: s.Name})
}

type createSectorRequest struct {
	Name string `json:"name" validate:"required"`
}

func (h *SectorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createSectorRequest
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
	s, err := h.uc.Create(r.Context(), req.Name)
	if err != nil {
		mapSectorError(w, err)
		return
	}
	response.Created(w, sectorResponse{ID: s.ID, Name: s.Name})
}

type updateSectorRequest struct {
	Name string `json:"name" validate:"required"`
}

func (h *SectorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "id is required")
		return
	}
	var req updateSectorRequest
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
	if err := h.uc.Update(r.Context(), id, req.Name); err != nil {
		mapSectorError(w, err)
		return
	}
	response.Message(w, http.StatusOK, "sector updated")
}

func (h *SectorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "id is required")
		return
	}
	if err := h.uc.SoftDelete(r.Context(), id); err != nil {
		mapSectorError(w, err)
		return
	}
	response.NoContent(w)
}

func mapSectorError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "validation failed")
	case errors.Is(err, domain.ErrSectorNotFound):
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "sector not found")
	case errors.Is(err, domain.ErrSectorNameTaken):
		response.Error(w, http.StatusConflict, response.CodeConflict, "sector name already exists")
	case errors.Is(err, domain.ErrSectorHasStocks):
		response.Error(w, http.StatusConflict, response.CodeConflict, "sector has stocks")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
	}
}
