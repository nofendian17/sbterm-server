// Package http provides HTTP handlers for the core service API.

package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/dto"
	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

type AdminHandler struct {
	uc usecase.AdminUsecase
	v  validator.Validator
}

func NewAdminHandler(uc usecase.AdminUsecase, v validator.Validator) *AdminHandler {
	return &AdminHandler{uc: uc, v: v}
}

// --- User management ---

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	page, limit = domain.NormalizePagination(page, limit)

	users, total, err := h.uc.ListUsers(r.Context(), page, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}

	resp := make([]dto.UserResponse, len(users))
	for i, u := range users {
		resp[i] = dto.ToUserResponse(u)
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	response.Paginated(w, resp, &response.MetaBody{
		Page:       page,
		Limit:      limit,
		TotalItems: total,
		TotalPages: totalPages,
	})
}

func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := h.uc.GetUser(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "user not found")
		return
	}
	response.OK(w, dto.ToUserResponse(user))
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.uc.DeleteUser(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "user not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	response.NoContent(w)
}

type expiryRequest struct {
	ExpiresAt  *string `json:"expires_at,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	ExtendDays *int    `json:"extend_days,omitempty" validate:"omitempty,gte=1"`
}

func (h *AdminHandler) SetExpiry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req expiryRequest
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

	if req.ExtendDays != nil {
		if err := h.uc.ExtendExpiry(r.Context(), id, *req.ExtendDays); err != nil {
			response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
			return
		}
		response.Message(w, http.StatusOK, "expiry extended")
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid expires_at format")
			return
		}
		expiresAt = &t
	}

	if err := h.uc.SetExpiry(r.Context(), id, expiresAt); err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	response.Message(w, http.StatusOK, "expiry updated")
}

// --- Role management ---

type createRoleRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description,omitempty"`
}

func (h *AdminHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.uc.ListRoles(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	response.OK(w, roles)
}

func (h *AdminHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	role, err := h.uc.GetRole(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "role not found")
		return
	}
	response.OK(w, role)
}

func (h *AdminHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req createRoleRequest
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

	role := domain.Role{
		Name:        req.Name,
		Description: req.Description,
	}
	created, err := h.uc.CreateRole(r.Context(), role)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	response.Created(w, created)
}

func (h *AdminHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.uc.DeleteRole(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrRoleNotFound) {
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "role not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	response.NoContent(w)
}

// --- Permission assignment ---

type permissionRequest struct {
	PermissionID string `json:"permission_id" validate:"required"`
}

func (h *AdminHandler) AssignPermissionToRole(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id")
	var req permissionRequest
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

	if err := h.uc.AssignPermissionToRole(r.Context(), roleID, req.PermissionID); err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	response.Message(w, http.StatusOK, "permission assigned")
}

func (h *AdminHandler) RevokePermissionFromRole(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id")
	permID := chi.URLParam(r, "permId")

	if err := h.uc.RevokePermissionFromRole(r.Context(), roleID, permID); err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	response.NoContent(w)
}

// --- User role assignment ---

type roleRequest struct {
	RoleID string `json:"role_id" validate:"required"`
}

func (h *AdminHandler) AssignRoleToUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	var req roleRequest
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

	if err := h.uc.AssignRoleToUser(r.Context(), userID, req.RoleID); err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	response.Message(w, http.StatusOK, "role assigned")
}

func (h *AdminHandler) RevokeRoleFromUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	roleID := chi.URLParam(r, "roleId")

	if err := h.uc.RevokeRoleFromUser(r.Context(), userID, roleID); err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	response.NoContent(w)
}
