package user

import (
	"encoding/json"
	"net/http"

	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/middleware"
	"github.com/nofendian17/sbterm/apps/core/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
)

type UserHandler struct {
	uc usecase.UserUsecase
}

func NewUserHandler(uc usecase.UserUsecase) *UserHandler {
	return &UserHandler{uc: uc}
}

type updateMeRequest struct {
	DisplayName string `json:"display_name"`
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

	response.OK(w, usecase.ToResponse(user))
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

	if err := h.uc.UpdateMe(r.Context(), userID, req.DisplayName); err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}

	response.OK(w, map[string]string{"message": "profile updated"})
}
