package stockbit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

const (
	userStreamPath         = "/stream/v3/user"
	streamAnnouncementPath = "/stream/announcement"
)

// UserStreamRequest is the POST body for the user stream endpoint.
type UserStreamRequest struct {
	Category     string `json:"category"`
	LastStreamID int64  `json:"last_stream_id,omitempty"`
	Limit        int    `json:"limit,omitempty"`
}

// UserStreamResponse is the user stream response: data.stream is the feed and
// data.pagination carries the cursor for older pages.
type UserStreamResponse struct {
	Message string `json:"message"`
	Data    struct {
		Stream              []StreamPost         `json:"stream"`
		Pagination          UserStreamPagination `json:"pagination"`
		InvalidWatchlistIDs []string             `json:"invalid_watchlist_ids"`
	} `json:"data"`
}

type UserStreamPagination struct {
	IsLastPage bool  `json:"is_last_page"`
	NextCursor int64 `json:"next_cursor"`
	Total      int   `json:"total"`
}

// StreamPost is one post in a user stream feed.
type StreamPost struct {
	StreamID       int64           `json:"stream_id"`
	TitleURL       string          `json:"title_url"`
	Title          string          `json:"title"`
	Content        string          `json:"content"`
	CreatedAt      string          `json:"created_at"`
	CreatedDisplay string          `json:"created_display"`
	UpdatedAt      string          `json:"updated_at"`
	User           StreamUser      `json:"user"`
	Status         StreamStatus    `json:"status"`
	TotalReplies   int             `json:"total_replies"`
	TotalLikes     int             `json:"total_likes"`
	Type           string          `json:"type"`
	ParentStreamID int64           `json:"parent_stream_id"`
	Reports        []StreamReport  `json:"reports"`
	Topics         []string        `json:"topics"`
	Summary        *StreamSummary  `json:"summary"`
	Reaction       *StreamReaction `json:"reaction"`
}

type StreamUser struct {
	UserID         int64  `json:"user_id"`
	IsAuthor       bool   `json:"is_author"`
	Username       string `json:"username"`
	Fullname       string `json:"fullname"`
	Avatar         string `json:"avatar"`
	IsVerified     bool   `json:"is_verified"`
	UserPrivilege  string `json:"user_privilege"`
	IsPro          bool   `json:"is_pro"`
	Country        string `json:"country"`
	VerifiedStatus string `json:"verified_status"`
}

type StreamStatus struct {
	IsPinned      bool `json:"is_pinned"`
	IsTrending    bool `json:"is_trending"`
	IsReposted    bool `json:"is_reposted"`
	IsLiked       bool `json:"is_liked"`
	IsSaved       bool `json:"is_saved"`
	IsFollowed    bool `json:"is_followed"`
	IsUnavailable bool `json:"is_unavailable"`
	IsJunk        bool `json:"is_junk"`
	IsSpam        bool `json:"is_spam"`
	IsViolation   bool `json:"is_violation"`
	IsDeleted     bool `json:"is_deleted"`
}

type StreamReport struct {
	Type string `json:"type"`
}

type StreamSummary struct {
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	KeyPoints    []string `json:"key_points"`
	KeyTakeaway  string   `json:"key_takeaway"`
	Model        string   `json:"model"`
	ModelVersion string   `json:"model_version"`
}

type StreamReaction struct {
	Reactions  []StreamReactionItem `json:"reactions"`
	Total      int                  `json:"total"`
	MyReaction any                  `json:"my_reaction"`
}

type StreamReactionItem struct {
	Reaction string `json:"reaction"`
	Total    int    `json:"total"`
}

// StreamAnnouncement is one attachment announced on a stream post.
type StreamAnnouncement struct {
	ID             int64  `json:"id"`
	CompanyID      int64  `json:"company_id"`
	PostedOn       string `json:"posted_on"`
	Headline       string `json:"headline"`
	Title          string `json:"title"`
	Attachment     string `json:"attachment"`
	RetrievedOn    string `json:"retrieved_on"`
	Symbol         string `json:"symbol"`
	Name           string `json:"name"`
	CompanyIconURL string `json:"company_icon_url"`
}

// StreamAnnouncementResponse is the announcement endpoint payload; data holds
// the attachments for a single stream post.
type StreamAnnouncementResponse struct {
	Message string               `json:"message"`
	Data    []StreamAnnouncement `json:"data"`
}

// GetStreamAnnouncement returns the announcement attachments for a stream post
// identified by its UUID. The access token is attached automatically.
func (c *Client) GetStreamAnnouncement(ctx context.Context, streamID string) (*StreamAnnouncementResponse, error) {
	path := streamAnnouncementPath + "/" + url.PathEscape(streamID)
	var out StreamAnnouncementResponse
	if err := c.Get(ctx, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetUserStream returns a user's stream feed, oldest-post-first paged by
// cursor: pass the previous response's next_cursor as lastStreamID to fetch
// older posts. The access token is attached automatically.
func (c *Client) GetUserStream(ctx context.Context, username string, req UserStreamRequest) (*UserStreamResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("stockbit: encode user stream request: %w", err)
	}
	path := userStreamPath + "/" + url.PathEscape(username)
	var out UserStreamResponse
	if err := c.Post(ctx, path, bytes.NewReader(body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}
