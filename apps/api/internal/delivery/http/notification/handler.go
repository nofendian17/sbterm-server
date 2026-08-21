package notification

import (
	"net/http"
	"strconv"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

const defaultLimit = 20

type NotificationHandler struct {
	uc usecase.NotificationUsecase
	v  validator.Validator
}

func NewNotificationHandler(uc usecase.NotificationUsecase, v validator.Validator) *NotificationHandler {
	return &NotificationHandler{uc: uc, v: v}
}

type notificationRequest struct {
	LastID int64    `json:"last_id" validate:"min=0"`
	Limit  int      `json:"limit" validate:"omitempty,min=1,max=25"`
	Types  []string `json:"types" validate:"omitempty,dive,oneof=NOTIF_TYPE_NEW_REPORT NOTIF_TYPE_NEWSFEED NOTIF_TYPE_COMPANY_PUBLIC_EXPOSE NOTIF_TYPE_COMPANY_SHAREHOLDING NOTIF_TYPE_COMPANY_DIVIDEND NOTIF_TYPE_COMPANY_CORP_ACTION NOTIF_TYPE_COMPANY_OTHERS"`
}

type notificationResponse struct {
	Unread int                `json:"unread"`
	Items  []notificationItem `json:"items"`
}

type notificationItem struct {
	ID      int64                    `json:"id"`
	Type    string                   `json:"type"`
	Avatar  string                   `json:"avatar"`
	Message string                   `json:"message"`
	Masks   []notificationMaskResp   `json:"masks,omitempty"`
	LinkTo  notificationLinkResponse `json:"link_to"`
	Created string                   `json:"created"`
	IsRead  bool                     `json:"is_read"`
}

type notificationMaskResp struct {
	Key  string `json:"key"`
	Text string `json:"text"`
	Tag  string `json:"tag"`
	Type string `json:"type"`
}

type notificationLinkResponse struct {
	Key   int64  `json:"key"`
	Value string `json:"value"`
}

func (h *NotificationHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	req := notificationRequest{Limit: defaultLimit}
	if raw := q.Get("last_id"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			response.ValidationError(w, "validation failed", map[string]string{"last_id": "must be a valid integer"})
			return
		}
		req.LastID = v
	}
	if raw := q.Get("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			response.ValidationError(w, "validation failed", map[string]string{"limit": "must be a valid integer"})
			return
		}
		req.Limit = v
	}
	req.Types = q["types"]

	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate notification params")
		return
	}

	list, err := h.uc.GetNotifications(r.Context(), req.LastID, req.Limit, req.Types)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get notifications")
		return
	}
	response.OK(w, toResponse(list))
}

func toResponse(l *domain.NotificationList) notificationResponse {
	out := notificationResponse{
		Unread: l.Unread,
		Items:  make([]notificationItem, 0, len(l.Items)),
	}
	for _, n := range l.Items {
		item := notificationItem{
			ID:      n.ID,
			Type:    n.Type,
			Avatar:  n.Avatar,
			Message: n.Message,
			LinkTo:  notificationLinkResponse{Key: n.LinkTo.Key, Value: n.LinkTo.Value},
			Created: n.Created,
			IsRead:  n.IsRead,
		}
		if len(n.Masks) > 0 {
			item.Masks = make([]notificationMaskResp, 0, len(n.Masks))
			for _, m := range n.Masks {
				item.Masks = append(item.Masks, notificationMaskResp{Key: m.Key, Text: m.Text, Tag: m.Tag, Type: m.Type})
			}
		}
		out.Items = append(out.Items, item)
	}
	return out
}
