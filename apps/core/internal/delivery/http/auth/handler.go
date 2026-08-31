package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
)

type AuthHandler struct {
	uc usecase.AuthUsecase
}

func NewAuthHandler(uc usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{uc: uc}
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	access, refresh, err := h.uc.Register(r.Context(), domain.RegisterInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		mapAuthError(w, err)
		return
	}

	response.Created(w, tokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	access, refresh, err := h.uc.Login(r.Context(), domain.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		mapAuthError(w, err)
		return
	}

	response.OK(w, tokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	access, refresh, err := h.uc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		mapAuthError(w, err)
		return
	}

	response.OK(w, tokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}

	if err := h.uc.Logout(r.Context(), req.RefreshToken); err != nil {
		mapAuthError(w, err)
		return
	}

	response.NoContent(w)
}

// mapAuthError maps domain errors to HTTP responses.
func mapAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "invalid email or password")
	case errors.Is(err, domain.ErrExpired):
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "account expired")
	case errors.Is(err, domain.ErrSuspended):
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "account suspended")
	case errors.Is(err, domain.ErrEmailTaken):
		response.Error(w, http.StatusConflict, response.CodeConflict, "email already registered")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
	}
}
