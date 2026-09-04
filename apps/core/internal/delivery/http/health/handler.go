// Package http provides HTTP handlers for the core service API.

package health

import (
	"net/http"

	"github.com/nofendian17/sbterm/apps/core/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
)

type HealthHandler struct {
	uc usecase.HealthUsecase
}

func NewHealthHandler(uc usecase.HealthUsecase) *HealthHandler {
	return &HealthHandler{uc: uc}
}

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Cache    string `json:"cache"`
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	status, err := h.uc.GetHealth(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to check health")
		return
	}

	httpStatus := http.StatusOK
	if !status.DBConnected || !status.CacheConnected {
		httpStatus = http.StatusServiceUnavailable
	}

	response.Success(w, httpStatus, healthResponse{
		Status:   status.Status,
		Database: dbStatus(status.DBConnected),
		Cache:    dbStatus(status.CacheConnected),
	})
}

func dbStatus(connected bool) string {
	if connected {
		return "up"
	}
	return "down"
}
