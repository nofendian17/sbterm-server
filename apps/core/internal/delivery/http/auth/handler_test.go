package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		setup      func(uc *mocks.MockAuthUsecase)
		wantCode   int
		wantTokens bool
	}{
		{
			name: "success",
			body: registerRequest{Email: "a@b.co", Password: "password123", DisplayName: "Test"},
			setup: func(uc *mocks.MockAuthUsecase) {
				uc.EXPECT().Register(gomock.Any(), domain.RegisterInput{
					Email:       "a@b.co",
					Password:    "password123",
					DisplayName: "Test",
				}).Return("access-token", "refresh-token", nil)
			},
			wantCode:   http.StatusCreated,
			wantTokens: true,
		},
		{
			name: "invalid body",
			body: "not json",
			setup: func(uc *mocks.MockAuthUsecase) {
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "email taken",
			body: registerRequest{Email: "a@b.co", Password: "password123", DisplayName: "Test"},
			setup: func(uc *mocks.MockAuthUsecase) {
				uc.EXPECT().Register(gomock.Any(), gomock.Any()).Return("", "", domain.ErrEmailTaken)
			},
			wantCode: http.StatusConflict,
		},
		{
			name: "internal error",
			body: registerRequest{Email: "a@b.co", Password: "password123", DisplayName: "Test"},
			setup: func(uc *mocks.MockAuthUsecase) {
				uc.EXPECT().Register(gomock.Any(), gomock.Any()).Return("", "", errors.New("db error"))
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "validation failed - empty fields",
			body: registerRequest{Email: "", Password: "", DisplayName: ""},
			setup: func(uc *mocks.MockAuthUsecase) {
			},
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name: "validation failed - weak password",
			body: registerRequest{Email: "a@b.co", Password: "short", DisplayName: "Test"},
			setup: func(uc *mocks.MockAuthUsecase) {
			},
			wantCode: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc := mocks.NewMockAuthUsecase(ctrl)
			tt.setup(uc)

			handler := NewAuthHandler(uc, validator.New())
			var body bytes.Buffer
			if s, ok := tt.body.(string); ok {
				body.WriteString(s)
			} else {
				json.NewEncoder(&body).Encode(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", &body)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			handler.Register(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
			if tt.wantTokens {
				var resp map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
				data := resp["data"].(map[string]any)
				assert.NotEmpty(t, data["access_token"])
				assert.NotEmpty(t, data["refresh_token"])
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		setup      func(uc *mocks.MockAuthUsecase)
		wantCode   int
		wantTokens bool
	}{
		{
			name: "success",
			body: loginRequest{Email: "a@b.co", Password: "password123"},
			setup: func(uc *mocks.MockAuthUsecase) {
				uc.EXPECT().Login(gomock.Any(), domain.LoginInput{
					Email:    "a@b.co",
					Password: "password123",
				}).Return("access-token", "refresh-token", nil)
			},
			wantCode:   http.StatusOK,
			wantTokens: true,
		},
		{
			name: "invalid credentials",
			body: loginRequest{Email: "a@b.co", Password: "wrong"},
			setup: func(uc *mocks.MockAuthUsecase) {
				uc.EXPECT().Login(gomock.Any(), gomock.Any()).Return("", "", domain.ErrInvalidCredentials)
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "account expired",
			body: loginRequest{Email: "a@b.co", Password: "password123"},
			setup: func(uc *mocks.MockAuthUsecase) {
				uc.EXPECT().Login(gomock.Any(), gomock.Any()).Return("", "", domain.ErrExpired)
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "account suspended",
			body: loginRequest{Email: "a@b.co", Password: "password123"},
			setup: func(uc *mocks.MockAuthUsecase) {
				uc.EXPECT().Login(gomock.Any(), gomock.Any()).Return("", "", domain.ErrSuspended)
			},
			wantCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc := mocks.NewMockAuthUsecase(ctrl)
			tt.setup(uc)

			handler := NewAuthHandler(uc, validator.New())
			var body bytes.Buffer
			if s, ok := tt.body.(string); ok {
				body.WriteString(s)
			} else {
				json.NewEncoder(&body).Encode(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", &body)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			handler.Login(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
			if tt.wantTokens {
				var resp map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
				data := resp["data"].(map[string]any)
				assert.NotEmpty(t, data["access_token"])
				assert.NotEmpty(t, data["refresh_token"])
			}
		})
	}
}

func TestAuthHandler_Refresh(t *testing.T) {
	tests := []struct {
		name     string
		body     any
		setup    func(uc *mocks.MockAuthUsecase)
		wantCode int
	}{
		{
			name: "success",
			body: refreshRequest{RefreshToken: "old-refresh"},
			setup: func(uc *mocks.MockAuthUsecase) {
				uc.EXPECT().Refresh(gomock.Any(), "old-refresh").Return("new-access", "new-refresh", nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name:     "invalid body",
			body:     "not json",
			setup:    func(uc *mocks.MockAuthUsecase) {},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "validation failed - empty refresh token",
			body:     refreshRequest{RefreshToken: ""},
			setup:    func(uc *mocks.MockAuthUsecase) {},
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid token",
			body: refreshRequest{RefreshToken: "bad"},
			setup: func(uc *mocks.MockAuthUsecase) {
				uc.EXPECT().Refresh(gomock.Any(), "bad").Return("", "", domain.ErrInvalidCredentials)
			},
			wantCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc := mocks.NewMockAuthUsecase(ctrl)
			tt.setup(uc)

			handler := NewAuthHandler(uc, validator.New())
			var body bytes.Buffer
			if s, ok := tt.body.(string); ok {
				body.WriteString(s)
			} else {
				json.NewEncoder(&body).Encode(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", &body)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			handler.Refresh(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(uc *mocks.MockAuthUsecase)
		wantCode int
	}{
		{
			name: "success",
			setup: func(uc *mocks.MockAuthUsecase) {
				uc.EXPECT().Logout(gomock.Any(), "refresh-token").Return(nil)
			},
			wantCode: http.StatusNoContent,
		},
		{
			name: "error",
			setup: func(uc *mocks.MockAuthUsecase) {
				uc.EXPECT().Logout(gomock.Any(), "bad").Return(errors.New("failed"))
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc := mocks.NewMockAuthUsecase(ctrl)
			tt.setup(uc)

			handler := NewAuthHandler(uc, validator.New())
			body, _ := json.Marshal(refreshRequest{RefreshToken: "refresh-token"})
			if tt.name == "error" {
				body, _ = json.Marshal(refreshRequest{RefreshToken: "bad"})
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			handler.Logout(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestAuthHandler_Logout_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockAuthUsecase(ctrl)

	handler := NewAuthHandler(uc, validator.New())
	body, _ := json.Marshal(refreshRequest{RefreshToken: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.Logout(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}
