package domain

// UserStreamData is the user stream feed data.
type UserStreamData struct {
	Stream              []StreamPost       `json:"stream"`
	Pagination          UserStreamPaginate `json:"pagination"`
	InvalidWatchlistIDs []string           `json:"invalid_watchlist_ids"`
}

type UserStreamPaginate struct {
	IsLastPage bool  `json:"is_last_page"`
	NextCursor int64 `json:"next_cursor"`
	Total      int   `json:"total"`
}

// StreamConversationData is a stream post plus one page of its replies.
type StreamConversationData struct {
	More    bool         `json:"more"`
	Prev    bool         `json:"prev"`
	Parent  StreamPost   `json:"parent"`
	Replies []StreamPost `json:"replies"`
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
