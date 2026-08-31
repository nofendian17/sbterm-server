package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/middleware"
	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
)

func TestUserHandler_GetMe(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		setup    func(uc *mocks.MockUserUsecase)
		wantCode int
	}{
		{
			name:   "success",
			userID: "u1",
			setup: func(uc *mocks.MockUserUsecase) {
				uc.EXPECT().GetMe(gomock.Any(), "u1").Return(domain.User{
					ID: "u1", Email: "a@b.co", DisplayName: "Test",
				}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name:     "unauthorized",
			userID:   "",
			setup:    func(uc *mocks.MockUserUsecase) {},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:   "internal error",
			userID: "u1",
			setup: func(uc *mocks.MockUserUsecase) {
				uc.EXPECT().GetMe(gomock.Any(), "u1").Return(domain.User{}, errors.New("db error"))
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc := mocks.NewMockUserUsecase(ctrl)
			tt.setup(uc)

			handler := NewUserHandler(uc)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), middleware.CtxUserID, tt.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()
			handler.GetMe(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestUserHandler_UpdateMe(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		body     any
		setup    func(uc *mocks.MockUserUsecase)
		wantCode int
	}{
		{
			name:   "success",
			userID: "u1",
			body:   updateMeRequest{DisplayName: "New Name"},
			setup: func(uc *mocks.MockUserUsecase) {
				uc.EXPECT().UpdateMe(gomock.Any(), "u1", "New Name").Return(nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name:     "unauthorized",
			userID:   "",
			body:     updateMeRequest{DisplayName: "New Name"},
			setup:    func(uc *mocks.MockUserUsecase) {},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "invalid body",
			userID:   "u1",
			body:     "not json",
			setup:    func(uc *mocks.MockUserUsecase) {},
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "internal error",
			userID: "u1",
			body:   updateMeRequest{DisplayName: "New Name"},
			setup: func(uc *mocks.MockUserUsecase) {
				uc.EXPECT().UpdateMe(gomock.Any(), "u1", "New Name").Return(errors.New("db error"))
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			uc := mocks.NewMockUserUsecase(ctrl)
			tt.setup(uc)

			handler := NewUserHandler(uc)
			var body bytes.Buffer
			if s, ok := tt.body.(string); ok {
				body.WriteString(s)
			} else {
				json.NewEncoder(&body).Encode(tt.body)
			}

			req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", &body)
			req.Header.Set("Content-Type", "application/json")
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), middleware.CtxUserID, tt.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()
			handler.UpdateMe(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}
