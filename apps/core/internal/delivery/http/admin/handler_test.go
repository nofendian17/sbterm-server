package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
	"github.com/nofendian17/sbterm/apps/core/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

func TestAdminHandler_ListUsers(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		setup    func(uc *mocks.MockAdminUsecase)
		wantCode int
	}{
		{
			name:  "success",
			query: "?page=1&limit=10",
			setup: func(uc *mocks.MockAdminUsecase) {
				uc.EXPECT().ListUsers(gomock.Any(), 1, 10).Return([]domain.User{
					{ID: "u1", Email: "a@b.co", DisplayName: "Test"},
				}, 1, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name:  "invalid page",
			query: "?page=abc",
			setup: func(uc *mocks.MockAdminUsecase) {
				uc.EXPECT().ListUsers(gomock.Any(), 1, 20).Return(nil, 0, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name:  "internal error",
			query: "",
			setup: func(uc *mocks.MockAdminUsecase) {
				uc.EXPECT().ListUsers(gomock.Any(), 1, 20).Return(nil, 0, errors.New("db error"))
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc := mocks.NewMockAdminUsecase(ctrl)
			tt.setup(uc)

			handler := NewAdminHandler(uc, validator.New())
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users"+tt.query, nil)
			rec := httptest.NewRecorder()
			handler.ListUsers(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestAdminHandler_GetUser(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		setup    func(uc *mocks.MockAdminUsecase)
		wantCode int
	}{
		{
			name:   "success",
			userID: "u1",
			setup: func(uc *mocks.MockAdminUsecase) {
				uc.EXPECT().GetUser(gomock.Any(), "u1").Return(domain.User{ID: "u1", Email: "a@b.co"}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name:   "not found",
			userID: "missing",
			setup: func(uc *mocks.MockAdminUsecase) {
				uc.EXPECT().GetUser(gomock.Any(), "missing").Return(domain.User{}, domain.ErrUserNotFound)
			},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc := mocks.NewMockAdminUsecase(ctrl)
			tt.setup(uc)

			handler := NewAdminHandler(uc, validator.New())
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.userID)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/"+tt.userID, nil)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rec := httptest.NewRecorder()
			handler.GetUser(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestAdminHandler_SuspendUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockAdminUsecase(ctrl)
	uc.EXPECT().SuspendUser(gomock.Any(), "u1").Return(nil)

	handler := NewAdminHandler(uc, validator.New())
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "u1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/u1/suspend", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	handler.SuspendUser(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAdminHandler_DeleteUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockAdminUsecase(ctrl)
	uc.EXPECT().DeleteUser(gomock.Any(), "u1").Return(nil)

	handler := NewAdminHandler(uc, validator.New())
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "u1")
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/u1", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	handler.DeleteUser(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestAdminHandler_ListRoles(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockAdminUsecase(ctrl)
	uc.EXPECT().ListRoles(gomock.Any()).Return([]domain.Role{
		{ID: "r1", Name: "admin"},
	}, nil)

	handler := NewAdminHandler(uc, validator.New())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/roles", nil)
	rec := httptest.NewRecorder()
	handler.ListRoles(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAdminHandler_CreateRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockAdminUsecase(ctrl)
	uc.EXPECT().CreateRole(gomock.Any(), gomock.Any()).Return(domain.Role{ID: "r1", Name: "moderator", Description: "Moderator"}, nil)

	handler := NewAdminHandler(uc, validator.New())
	body, _ := json.Marshal(createRoleRequest{Name: "moderator", Description: "Moderator"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.CreateRole(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestAdminHandler_CreateRole_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockAdminUsecase(ctrl)

	handler := NewAdminHandler(uc, validator.New())
	body, _ := json.Marshal(createRoleRequest{Name: "", Description: "desc"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.CreateRole(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestAdminHandler_DeleteRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockAdminUsecase(ctrl)
	uc.EXPECT().DeleteRole(gomock.Any(), "r1").Return(nil)

	handler := NewAdminHandler(uc, validator.New())
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "r1")
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/roles/r1", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	handler.DeleteRole(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestAdminHandler_SetExpiry(t *testing.T) {
	tests := []struct {
		name     string
		body     any
		setup    func(uc *mocks.MockAdminUsecase)
		wantCode int
	}{
		{
			name: "set expiry",
			body: expiryRequest{ExpiresAt: ptrString("2026-12-31T00:00:00Z")},
			setup: func(uc *mocks.MockAdminUsecase) {
				uc.EXPECT().SetExpiry(gomock.Any(), "u1", gomock.Any()).Return(nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name: "extend expiry",
			body: expiryRequest{ExtendDays: ptrInt(7)},
			setup: func(uc *mocks.MockAdminUsecase) {
				uc.EXPECT().ExtendExpiry(gomock.Any(), "u1", 7).Return(nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name:     "invalid body",
			body:     "not json",
			setup:    func(uc *mocks.MockAdminUsecase) {},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "validation failed - invalid expires_at format",
			body:     expiryRequest{ExpiresAt: ptrString("not-a-date")},
			setup:    func(uc *mocks.MockAdminUsecase) {},
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "validation failed - extend_days less than 1",
			body:     expiryRequest{ExtendDays: ptrInt(0)},
			setup:    func(uc *mocks.MockAdminUsecase) {},
			wantCode: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc := mocks.NewMockAdminUsecase(ctrl)
			tt.setup(uc)

			handler := NewAdminHandler(uc, validator.New())
			var body bytes.Buffer
			if s, ok := tt.body.(string); ok {
				body.WriteString(s)
			} else {
				json.NewEncoder(&body).Encode(tt.body)
			}

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", "u1")
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/u1/expiry", &body)
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rec := httptest.NewRecorder()
			handler.SetExpiry(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestAdminHandler_AssignRoleToUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockAdminUsecase(ctrl)
	uc.EXPECT().AssignRoleToUser(gomock.Any(), "u1", "r1").Return(nil)

	handler := NewAdminHandler(uc, validator.New())
	body, _ := json.Marshal(roleRequest{RoleID: "r1"})
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "u1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/u1/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	handler.AssignRoleToUser(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAdminHandler_AssignRoleToUser_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockAdminUsecase(ctrl)

	handler := NewAdminHandler(uc, validator.New())
	body, _ := json.Marshal(roleRequest{RoleID: ""})
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "u1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/u1/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	handler.AssignRoleToUser(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestAdminHandler_RevokeRoleFromUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockAdminUsecase(ctrl)
	uc.EXPECT().RevokeRoleFromUser(gomock.Any(), "u1", "r1").Return(nil)

	handler := NewAdminHandler(uc, validator.New())
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "u1")
	rctx.URLParams.Add("roleId", "r1")
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/u1/roles/r1", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	handler.RevokeRoleFromUser(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func ptrString(s string) *string { return &s }
func ptrInt(n int) *int          { return &n }

// Ensure the handler uses the correct usecase interface.
var _ usecase.AdminUsecase = (*mocks.MockAdminUsecase)(nil)
