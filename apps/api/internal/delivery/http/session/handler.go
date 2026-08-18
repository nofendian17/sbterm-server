package session

import (
	"net/http"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
)

type MarketSessionHandler struct {
	uc usecase.MarketSessionUsecase
}

func NewMarketSessionHandler(uc usecase.MarketSessionUsecase) *MarketSessionHandler {
	return &MarketSessionHandler{uc: uc}
}

type marketSessionResponse struct {
	Datetime string                       `json:"datetime"`
	FCA      marketSessionSegmentResponse `json:"fca"`
	Regular  marketSessionSegmentResponse `json:"regular"`
}

type marketSessionSegmentResponse struct {
	StateName      string `json:"state_name"`
	IsLastSession  bool   `json:"is_last_session"`
	IsEndOfDay     bool   `json:"is_end_of_day"`
	StateStartTime string `json:"state_start_time"`
	StateEndTime   string `json:"state_end_time"`
	TimeLeft       string `json:"time_left"`
}

func (h *MarketSessionHandler) MarketSession(w http.ResponseWriter, r *http.Request) {
	s, err := h.uc.GetMarketSession(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get market session")
		return
	}
	response.OK(w, marketSessionResponse{
		Datetime: s.Datetime,
		FCA:      toResponse(s.FCA),
		Regular:  toResponse(s.Regular),
	})
}

func toResponse(s domain.MarketSessionSegment) marketSessionSegmentResponse {
	return marketSessionSegmentResponse{
		StateName:      s.StateName,
		IsLastSession:  s.IsLastSession,
		IsEndOfDay:     s.IsEndOfDay,
		StateStartTime: s.StateStartTime,
		StateEndTime:   s.StateEndTime,
		TimeLeft:       s.TimeLeft,
	}
}
