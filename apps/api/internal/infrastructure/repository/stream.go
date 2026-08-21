package repository

import (
	"context"
	"errors"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// StreamRepository fetches a user's stream feed from the Stockbit API.
type StreamRepository struct {
	client *stockbit.Client
}

func NewStreamRepository(client *stockbit.Client) *StreamRepository {
	return &StreamRepository{client: client}
}

func (r *StreamRepository) GetUserStream(ctx context.Context, username, category string, lastStreamID int64, limit int) (*domain.UserStreamData, error) {
	resp, err := r.client.GetUserStream(ctx, username, stockbit.UserStreamRequest{
		Category:     category,
		LastStreamID: lastStreamID,
		Limit:        limit,
	})
	if err != nil {
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg}
		}
		return nil, err
	}
	posts := make([]domain.StreamPost, 0, len(resp.Data.Stream))
	for _, p := range resp.Data.Stream {
		posts = append(posts, toDomainStreamPost(p))
	}
	return &domain.UserStreamData{
		Stream: posts,
		Pagination: domain.UserStreamPaginate{
			IsLastPage: resp.Data.Pagination.IsLastPage,
			NextCursor: resp.Data.Pagination.NextCursor,
			Total:      resp.Data.Pagination.Total,
		},
		InvalidWatchlistIDs: resp.Data.InvalidWatchlistIDs,
	}, nil
}

func (r *StreamRepository) GetStreamConversation(ctx context.Context, streamID string) (*domain.StreamConversationData, error) {
	resp, err := r.client.GetStreamConversation(ctx, streamID)
	if err != nil {
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg}
		}
		return nil, err
	}
	c := resp.Data.Conversation
	replies := make([]domain.StreamPost, 0, len(c.Replies))
	for _, p := range c.Replies {
		replies = append(replies, toDomainStreamPost(p))
	}
	return &domain.StreamConversationData{
		More:    c.More,
		Prev:    c.Prev,
		Parent:  toDomainStreamPost(c.Parent),
		Replies: replies,
	}, nil
}

func toDomainStreamPost(p stockbit.StreamPost) domain.StreamPost {
	return domain.StreamPost{
		StreamID:       p.StreamID,
		TitleURL:       p.TitleURL,
		Title:          p.Title,
		Content:        p.Content,
		CreatedAt:      p.CreatedAt,
		CreatedDisplay: p.CreatedDisplay,
		UpdatedAt:      p.UpdatedAt,
		User: domain.StreamUser{
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
		Status: domain.StreamStatus{
			IsPinned:      p.Status.IsPinned,
			IsTrending:    p.Status.IsTrending,
			IsReposted:    p.Status.IsReposted,
			IsLiked:       p.Status.IsLiked,
			IsSaved:       p.Status.IsSaved,
			IsFollowed:    p.Status.IsFollowed,
			IsUnavailable: p.Status.IsUnavailable,
			IsJunk:        p.Status.IsJunk,
			IsSpam:        p.Status.IsSpam,
			IsViolation:   p.Status.IsViolation,
			IsDeleted:     p.Status.IsDeleted,
		},
		TotalReplies:   p.TotalReplies,
		TotalLikes:     p.TotalLikes,
		Type:           p.Type,
		ParentStreamID: p.ParentStreamID,
		Reports:        toStreamReports(p.Reports),
		Topics:         p.Topics,
		Summary:        toStreamSummary(p.Summary),
		Reaction:       toStreamReaction(p.Reaction),
	}
}

func (r *StreamRepository) GetStreamAnnouncement(ctx context.Context, streamID string) ([]domain.StreamAnnouncement, error) {
	resp, err := r.client.GetStreamAnnouncement(ctx, streamID)
	if err != nil {
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg}
		}
		return nil, err
	}
	out := make([]domain.StreamAnnouncement, 0, len(resp.Data))
	for _, a := range resp.Data {
		out = append(out, domain.StreamAnnouncement{
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
	return out, nil
}

func toStreamReports(in []stockbit.StreamReport) []domain.StreamReport {
	out := make([]domain.StreamReport, 0, len(in))
	for _, r := range in {
		out = append(out, domain.StreamReport{Type: r.Type})
	}
	return out
}

func toStreamSummary(in *stockbit.StreamSummary) *domain.StreamSummary {
	if in == nil {
		return nil
	}
	return &domain.StreamSummary{
		Title:        in.Title,
		Summary:      in.Summary,
		KeyPoints:    in.KeyPoints,
		KeyTakeaway:  in.KeyTakeaway,
		Model:        in.Model,
		ModelVersion: in.ModelVersion,
	}
}

func toStreamReaction(in *stockbit.StreamReaction) *domain.StreamReaction {
	if in == nil {
		return nil
	}
	out := &domain.StreamReaction{
		Total:      in.Total,
		MyReaction: in.MyReaction,
		Reactions:  make([]domain.StreamReactionItem, 0, len(in.Reactions)),
	}
	for _, r := range in.Reactions {
		out.Reactions = append(out.Reactions, domain.StreamReactionItem{Reaction: r.Reaction, Total: r.Total})
	}
	return out
}

var _ repository.StreamRepository = (*StreamRepository)(nil)
