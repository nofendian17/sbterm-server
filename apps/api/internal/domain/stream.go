package domain

// UserStreamData is the user stream feed data.
type UserStreamData struct {
	Stream              []StreamPost
	Pagination          UserStreamPaginate
	InvalidWatchlistIDs []string
}

type UserStreamPaginate struct {
	IsLastPage bool
	NextCursor int64
	Total      int
}

// StreamConversationData is a stream post plus one page of its replies.
type StreamConversationData struct {
	More    bool
	Prev    bool
	Parent  StreamPost
	Replies []StreamPost
}

// StreamPost is one post in a user stream feed.
type StreamPost struct {
	StreamID       int64
	TitleURL       string
	Title          string
	Content        string
	CreatedAt      string
	CreatedDisplay string
	UpdatedAt      string
	User           StreamUser
	Status         StreamStatus
	TotalReplies   int
	TotalLikes     int
	Type           string
	ParentStreamID int64
	Reports        []StreamReport
	Topics         []string
	Summary        *StreamSummary
	Reaction       *StreamReaction
}

type StreamUser struct {
	UserID         int64
	IsAuthor       bool
	Username       string
	Fullname       string
	Avatar         string
	IsVerified     bool
	UserPrivilege  string
	IsPro          bool
	Country        string
	VerifiedStatus string
}

type StreamStatus struct {
	IsPinned      bool
	IsTrending    bool
	IsReposted    bool
	IsLiked       bool
	IsSaved       bool
	IsFollowed    bool
	IsUnavailable bool
	IsJunk        bool
	IsSpam        bool
	IsViolation   bool
	IsDeleted     bool
}

type StreamReport struct {
	Type string
}

type StreamSummary struct {
	Title        string
	Summary      string
	KeyPoints    []string
	KeyTakeaway  string
	Model        string
	ModelVersion string
}

type StreamReaction struct {
	Reactions  []StreamReactionItem
	Total      int
	MyReaction any
}

type StreamReactionItem struct {
	Reaction string
	Total    int
}

// StreamAnnouncement is one attachment announced on a stream post.
type StreamAnnouncement struct {
	ID             int64
	CompanyID      int64
	PostedOn       string
	Headline       string
	Title          string
	Attachment     string
	RetrievedOn    string
	Symbol         string
	Name           string
	CompanyIconURL string
}
