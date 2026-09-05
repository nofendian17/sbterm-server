package sector

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

func withID(r *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestSectorHandler_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockSectorUsecase(ctrl)
	uc.EXPECT().List(gomock.Any()).
		Return([]domain.Sector{{ID: "s1", Name: "Financials"}}, nil)

	handler := NewSectorHandler(uc, validator.New())
	rec := httptest.NewRecorder()
	handler.List(rec, httptest.NewRequest(http.MethodGet, "/api/v1/sectors", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	var env struct {
		Success bool `json:"success"`
		Data    []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.Len(t, env.Data, 1)
	assert.Equal(t, "Financials", env.Data[0].Name)
}

func TestSectorHandler_Create_Duplicate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockSectorUsecase(ctrl)
	uc.EXPECT().Create(gomock.Any(), "Financials").
		Return(domain.Sector{}, domain.ErrSectorNameTaken)

	handler := NewSectorHandler(uc, validator.New())
	body, _ := json.Marshal(createSectorRequest{Name: "Financials"})
	rec := httptest.NewRecorder()
	handler.Create(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/sectors", bytes.NewReader(body)))

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestSectorHandler_Delete_HasStocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockSectorUsecase(ctrl)
	uc.EXPECT().SoftDelete(gomock.Any(), "s1").
		Return(domain.ErrSectorHasStocks)

	handler := NewSectorHandler(uc, validator.New())
	req := withID(httptest.NewRequest(http.MethodDelete, "/api/v1/admin/sectors/s1", nil), "s1")
	rec := httptest.NewRecorder()
	handler.Delete(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}
