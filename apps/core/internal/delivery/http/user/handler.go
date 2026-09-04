// Package http provides HTTP handlers for the core service API.

package user

import (
	"encoding/json"
	"net/http"

	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/dto"
	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/middleware"
	"github.com/nofendian17/sbterm/apps/core/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

type UserHandler struct {
	uc usecase.UserUsecase
	v  validator.Validator
}

func NewUserHandler(uc usecase.UserUsecase, v validator.Validator) *UserHandler {
	return &UserHandler{uc: uc, v: v}
}

type updateMeRequest struct {
	DisplayName string `json:"display_name" validate:"required"`
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "unauthorized")
		return
	}

	user, err := h.uc.GetMe(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}

	response.OK(w, dto.ToUserResponse(user))
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "unauthorized")
		return
	}

	var req updateMeRequest
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

	if err := h.uc.UpdateMe(r.Context(), userID, req.DisplayName); err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}

	response.OK(w, map[string]string{"message": "profile updated"})
}
