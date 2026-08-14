package stream

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/usecase"
	"github.com/nofendian17/sbterm-server/pkg/response"
	"github.com/nofendian17/sbterm-server/pkg/validator"
)

// Default values applied when the corresponding query param is omitted.
const (
	defaultCategory = "STREAM_CATEGORY_MAIN_IDEAS"
	defaultLimit    = 20
)

type StreamHandler struct {
	uc usecase.StreamUsecase
	v  validator.Validator
}

func NewStreamHandler(uc usecase.StreamUsecase, v validator.Validator) *StreamHandler {
	return &StreamHandler{uc: uc, v: v}
}

type userStreamRequest struct {
	Category     string `validate:"omitempty,oneof=STREAM_CATEGORY_MAIN_IDEAS STREAM_CATEGORY_NEWS"`
	LastStreamID int64
	Limit        int `validate:"min=1"`
}

type streamPostResponse struct {
	StreamID       int64                   `json:"stream_id"`
	TitleURL       string                  `json:"title_url"`
	Title          string                  `json:"title"`
	Content        string                  `json:"content"`
	CreatedAt      string                  `json:"created_at"`
	CreatedDisplay string                  `json:"created_display"`
	UpdatedAt      string                  `json:"updated_at"`
	User           streamUserResponse      `json:"user"`
	Status         streamStatusResponse    `json:"status"`
	TotalReplies   int                     `json:"total_replies"`
	TotalLikes     int                     `json:"total_likes"`
	Type           string                  `json:"type"`
	ParentStreamID int64                   `json:"parent_stream_id"`
	Reports        []streamReportResponse  `json:"reports"`
	Topics         []string                `json:"topics"`
	Summary        *streamSummaryResponse  `json:"summary"`
	Reaction       *streamReactionResponse `json:"reaction"`
}

type streamUserResponse struct {
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

type streamStatusResponse struct {
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

type streamReportResponse struct {
	Type string `json:"type"`
}

type streamSummaryResponse struct {
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	KeyPoints    []string `json:"key_points"`
	KeyTakeaway  string   `json:"key_takeaway"`
	Model        string   `json:"model"`
	ModelVersion string   `json:"model_version"`
}

type streamReactionResponse struct {
	Reactions  []streamReactionItemResponse `json:"reactions"`
	Total      int                          `json:"total"`
	MyReaction any                          `json:"my_reaction"`
}

type streamReactionItemResponse struct {
	Reaction string `json:"reaction"`
	Total    int    `json:"total"`
}

type userStreamPaginateResponse struct {
	IsLastPage bool  `json:"is_last_page"`
	NextCursor int64 `json:"next_cursor"`
	Total      int   `json:"total"`
}

type userStreamResponse struct {
	Stream              []streamPostResponse       `json:"stream"`
	Pagination          userStreamPaginateResponse `json:"pagination"`
	InvalidWatchlistIDs []string                   `json:"invalid_watchlist_ids"`
}

func (h *StreamHandler) UserStream(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		response.ValidationError(w, "validation failed", map[string]string{"username": "is required"})
		return
	}

	req := userStreamRequest{
		Category:     r.URL.Query().Get("category"),
		LastStreamID: 0,
		Limit:        0,
	}
	if req.Category == "" {
		req.Category = defaultCategory
	}

	lastStreamID, err := parseInt64Query(r.URL.Query().Get("last_stream_id"))
	if err != nil {
		response.ValidationError(w, "validation failed", map[string]string{"last_stream_id": "must be a valid integer"})
		return
	}
	req.LastStreamID = lastStreamID

	limit, err := parseIntQuery(r.URL.Query().Get("limit"), defaultLimit)
	if err != nil {
		response.ValidationError(w, "validation failed", map[string]string{"limit": "must be a valid integer"})
		return
	}
	req.Limit = limit

	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate user stream params")
		return
	}

	data, err := h.uc.GetUserStream(r.Context(), username, req.Category, req.LastStreamID, req.Limit)
	if err != nil {
		var upErr *domain.UpstreamError
		if errors.As(err, &upErr) && upErr.Status == http.StatusBadRequest {
			response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "no user stream data for the requested parameters")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get user stream")
		return
	}
	response.OK(w, toUserStreamResponse(data))
}

type streamAnnouncementResponse struct {
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

func (h *StreamHandler) StreamAnnouncement(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "stream_id")
	if streamID == "" {
		response.ValidationError(w, "validation failed", map[string]string{"stream_id": "is required"})
		return
	}

	announcements, err := h.uc.GetStreamAnnouncement(r.Context(), streamID)
	if err != nil {
		var upErr *domain.UpstreamError
		if errors.As(err, &upErr) && upErr.Status == http.StatusBadRequest {
			response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "no announcement data for the requested stream")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to get stream announcement")
		return
	}
	response.OK(w, toStreamAnnouncementResponses(announcements))
}

func toStreamAnnouncementResponses(in []domain.StreamAnnouncement) []streamAnnouncementResponse {
	out := make([]streamAnnouncementResponse, 0, len(in))
	for _, a := range in {
		out = append(out, streamAnnouncementResponse{
			ID:             a.ID,
			CompanyID:      a.CompanyID,
			PostedOn:       a.PostedOn,
			Headline:       a.Headline,
			Title:          a.Title,
			Attachment:     a.Attachment,
			RetrievedOn:    a.RetrievedOn,
			Symbol:         a.Symbol,
			Name:           a.Name,
			CompanyIconURL: a.CompanyIconURL,
		})
	}
	return out
}

func parseIntQuery(s string, def int) (int, error) {
	if s == "" {
		return def, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func parseInt64Query(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func toUserStreamResponse(d *domain.UserStreamData) userStreamResponse {
	out := userStreamResponse{
		Stream: make([]streamPostResponse, 0, len(d.Stream)),
		Pagination: userStreamPaginateResponse{
			IsLastPage: d.Pagination.IsLastPage,
			NextCursor: d.Pagination.NextCursor,
			Total:      d.Pagination.Total,
		},
		InvalidWatchlistIDs: d.InvalidWatchlistIDs,
	}
	for _, p := range d.Stream {
		out.Stream = append(out.Stream, streamPostResponse{
			StreamID:       p.StreamID,
			TitleURL:       p.TitleURL,
			Title:          p.Title,
			Content:        p.Content,
			CreatedAt:      p.CreatedAt,
			CreatedDisplay: p.CreatedDisplay,
			UpdatedAt:      p.UpdatedAt,
			User: streamUserResponse{
				UserID:         p.User.UserID,
				IsAuthor:       p.User.IsAuthor,
				Username:       p.User.Username,
				Fullname:       p.User.Fullname,
				Avatar:         p.User.Avatar,
				IsVerified:     p.User.IsVerified,
				UserPrivilege:  p.User.UserPrivilege,
				IsPro:          p.User.IsPro,
				Country:        p.User.Country,
				VerifiedStatus: p.User.VerifiedStatus,
			},
			Status:         toStreamStatusResponse(p.Status),
			TotalReplies:   p.TotalReplies,
			TotalLikes:     p.TotalLikes,
			Type:           p.Type,
			ParentStreamID: p.ParentStreamID,
			Reports:        toStreamReportResponses(p.Reports),
			Topics:         p.Topics,
			Summary:        toStreamSummaryResponse(p.Summary),
			Reaction:       toStreamReactionResponse(p.Reaction),
		})
	}
	return out
}

func toStreamStatusResponse(s domain.StreamStatus) streamStatusResponse {
	return streamStatusResponse{
		IsPinned:      s.IsPinned,
		IsTrending:    s.IsTrending,
		IsReposted:    s.IsReposted,
		IsLiked:       s.IsLiked,
		IsSaved:       s.IsSaved,
		IsFollowed:    s.IsFollowed,
		IsUnavailable: s.IsUnavailable,
		IsJunk:        s.IsJunk,
		IsSpam:        s.IsSpam,
		IsViolation:   s.IsViolation,
		IsDeleted:     s.IsDeleted,
	}
}

func toStreamReportResponses(in []domain.StreamReport) []streamReportResponse {
	out := make([]streamReportResponse, 0, len(in))
	for _, r := range in {
		out = append(out, streamReportResponse{Type: r.Type})
	}
	return out
}

func toStreamSummaryResponse(in *domain.StreamSummary) *streamSummaryResponse {
	if in == nil {
		return nil
	}
	return &streamSummaryResponse{
		Title:        in.Title,
		Summary:      in.Summary,
		KeyPoints:    in.KeyPoints,
		KeyTakeaway:  in.KeyTakeaway,
		Model:        in.Model,
		ModelVersion: in.ModelVersion,
	}
}

func toStreamReactionResponse(in *domain.StreamReaction) *streamReactionResponse {
	if in == nil {
		return nil
	}
	out := &streamReactionResponse{
		Total:      in.Total,
		MyReaction: in.MyReaction,
		Reactions:  make([]streamReactionItemResponse, 0, len(in.Reactions)),
	}
	for _, r := range in.Reactions {
		out.Reactions = append(out.Reactions, streamReactionItemResponse{Reaction: r.Reaction, Total: r.Total})
	}
	return out
}
