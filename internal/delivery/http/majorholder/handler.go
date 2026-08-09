package majorholder

import (
	"net/http"
	"strconv"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/response"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

type MajorHolderHandler struct {
	uc usecase.MajorHolderUsecase
	v  validator.Validator
}

func NewMajorHolderHandler(uc usecase.MajorHolderUsecase, v validator.Validator) *MajorHolderHandler {
	return &MajorHolderHandler{uc: uc, v: v}
}

type majorHolderRequest struct {
	Symbols    string `json:"symbols" validate:"required"`
	ActionType string `json:"action_type" validate:"omitempty,oneof=ACTION_TYPE_UNSPECIFIED ACTION_TYPE_BUY ACTION_TYPE_SELL ACTION_TYPE_CROSS ACTION_TYPE_TRANSFER ACTION_TYPE_CORPACTION"`
	SourceType string `json:"source_type" validate:"omitempty,oneof=SOURCE_TYPE_UNSPECIFIED SOURCE_TYPE_KSEI SOURCE_TYPE_IDX"`
	Page       int    `json:"page" validate:"omitempty"`
	Limit      int    `json:"limit" validate:"omitempty"`
}

type majorHolderResponse struct {
	IsMore   bool                      `json:"is_more"`
	Movement []majorHolderItemResponse `json:"movement"`
}

type majorHolderItemResponse struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Symbol         string                 `json:"symbol"`
	Date           string                 `json:"date"`
	Previous       majorHolderValueChange `json:"previous"`
	Current        majorHolderValueChange `json:"current"`
	Changes        majorHolderValueChange `json:"changes"`
	Marker         string                 `json:"marker"`
	IsPosted       bool                   `json:"is_posted"`
	CMHID          string                 `json:"cmh_id"`
	Nationality    string                 `json:"nationality"`
	ActionType     string                 `json:"action_type"`
	DataSource     majorHolderDataSource  `json:"data_source"`
	PriceFormatted string                 `json:"price_formatted"`
	BrokerDetail   majorHolderBroker      `json:"broker_detail"`
	Badges         []string               `json:"badges"`
}

type majorHolderValueChange struct {
	Value          string `json:"value"`
	Percentage     string `json:"percentage"`
	FormattedValue string `json:"formatted_value"`
}

type majorHolderDataSource struct {
	Label string `json:"label"`
	Type  string `json:"type"`
}

type majorHolderBroker struct {
	Code  string `json:"code"`
	Group string `json:"group"`
}

func (h *MajorHolderHandler) MajorHolder(w http.ResponseWriter, r *http.Request) {
	req := majorHolderRequest{
		Symbols:    r.URL.Query().Get("symbols"),
		ActionType: r.URL.Query().Get("action_type"),
		SourceType: r.URL.Query().Get("source_type"),
	}
	if v := r.URL.Query().Get("page"); v != "" {
		req.Page, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		req.Limit, _ = strconv.Atoi(v)
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate major holder params")
		return
	}

	data, err := h.uc.GetMajorHolder(r.Context(), req.Symbols, req.ActionType, req.SourceType, req.Page, req.Limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get major holder data")
		return
	}
	response.OK(w, toResponse(data))
}

func toResponse(d *domain.MajorHolderData) majorHolderResponse {
	out := majorHolderResponse{
		IsMore:   d.IsMore,
		Movement: make([]majorHolderItemResponse, 0, len(d.Movement)),
	}
	for _, m := range d.Movement {
		out.Movement = append(out.Movement, majorHolderItemResponse{
			ID:     m.ID,
			Name:   m.Name,
			Symbol: m.Symbol,
			Date:   m.Date,
			Previous: majorHolderValueChange{
				Value:          m.Previous.Value,
				Percentage:     m.Previous.Percentage,
				FormattedValue: m.Previous.FormattedValue,
			},
			Current: majorHolderValueChange{
				Value:          m.Current.Value,
				Percentage:     m.Current.Percentage,
				FormattedValue: m.Current.FormattedValue,
			},
			Changes: majorHolderValueChange{
				Value:          m.Changes.Value,
				Percentage:     m.Changes.Percentage,
				FormattedValue: m.Changes.FormattedValue,
			},
			Marker:         m.Marker,
			IsPosted:       m.IsPosted,
			CMHID:          m.CMHID,
			Nationality:    m.Nationality,
			ActionType:     m.ActionType,
			DataSource:     majorHolderDataSource{Label: m.DataSource.Label, Type: m.DataSource.Type},
			PriceFormatted: m.PriceFormatted,
			BrokerDetail:   majorHolderBroker{Code: m.BrokerDetail.Code, Group: m.BrokerDetail.Group},
			Badges:         m.Badges,
		})
	}
	return out
}
