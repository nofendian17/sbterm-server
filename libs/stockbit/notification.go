package stockbit

import (
	"context"
	"net/url"
	"strconv"
)

const notificationPath = "/notification"

// NotificationType selects which notification kinds the endpoint returns.
type NotificationType string

const (
	NotifTypeNewReport           NotificationType = "NOTIF_TYPE_NEW_REPORT"
	NotifTypeNewsfeed            NotificationType = "NOTIF_TYPE_NEWSFEED"
	NotifTypeCompanyPublicExpose NotificationType = "NOTIF_TYPE_COMPANY_PUBLIC_EXPOSE"
	NotifTypeCompanyShareholding NotificationType = "NOTIF_TYPE_COMPANY_SHAREHOLDING"
	NotifTypeCompanyDividend     NotificationType = "NOTIF_TYPE_COMPANY_DIVIDEND"
	NotifTypeCompanyCorpAction   NotificationType = "NOTIF_TYPE_COMPANY_CORP_ACTION"
	NotifTypeCompanyOthers       NotificationType = "NOTIF_TYPE_COMPANY_OTHERS"
)

// NotificationRequest selects a page of notifications. LastID is the cursor
// (0 = newest); upstream caps Limit at 25; nil/empty Types returns all types.
type NotificationRequest struct {
	LastID int64
	Limit  int
	Types  []NotificationType
}

// NotificationResponse is the notification response: data.result is the page,
// data.unread the unread count.
type NotificationResponse struct {
	Message string `json:"message"`
	Data    struct {
		Unread int            `json:"unread"`
		Result []Notification `json:"result"`
	} `json:"data"`
}

// Notification is one entry. All known types share this payload shape.
type Notification struct {
	ID      int64            `json:"id"`
	Type    NotificationType `json:"type"`
	Data    NotificationData `json:"data"`
	Created string           `json:"created"` // RFC3339 timestamp from upstream
	IsRead  bool             `json:"is_read"`
}

type NotificationData struct {
	Avatar      string                    `json:"avatar"`
	Message     string                    `json:"message"` // template with %key% placeholders resolved via MessageMask
	MessageMask []NotificationMessageMask `json:"message_mask"`
	LinkTo      NotificationLinkTo        `json:"link_to"`
}

// NotificationMessageMask maps one %key% placeholder in Message to its payload.
type NotificationMessageMask struct {
	Key     string                  `json:"key"`
	Payload NotificationMaskPayload `json:"payload"`
}

type NotificationMaskPayload struct {
	ID     int64  `json:"id"`
	Tag    string `json:"tag"`  // e.g. PAYLOAD_MASK_TAG_BOLD
	Text   string `json:"text"` // display text (symbol, company or user name)
	Type   string `json:"type"` // e.g. PAYLOAD_MASK_TYPE_COMPANY / _USER
	Actors []any  `json:"actors"`
}

// NotificationLinkTo is the navigation target of the notification.
type NotificationLinkTo struct {
	Key   int64  `json:"key"`
	Value string `json:"value"`
}

// GetNotifications returns one page of notifications for the authenticated
// user. The access token is attached automatically.
func (c *Client) GetNotifications(ctx context.Context, req NotificationRequest) (*NotificationResponse, error) {
	q := url.Values{}
	if req.LastID > 0 {
		q.Set("last_id", strconv.FormatInt(req.LastID, 10))
	}
	if req.Limit > 0 {
		q.Set("limit", strconv.Itoa(req.Limit))
	}
	for _, t := range req.Types {
		q.Add("types", string(t))
	}
	var out NotificationResponse
	if err := c.Get(ctx, notificationPath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
