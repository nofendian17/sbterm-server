package repository

import (
	"context"
	"errors"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// SearchRepository fetches symbol search results from the Stockbit API.
type SearchRepository struct {
	client *stockbit.Client
}

func NewSearchRepository(client *stockbit.Client) *SearchRepository {
	return &SearchRepository{client: client}
}

func (r *SearchRepository) GetSearch(ctx context.Context, keyword string, page int, typ string) (*domain.SearchResult, error) {
	resp, err := r.client.GetSearch(ctx, keyword, page, typ)
	if err != nil {
		// Translate the client's typed status error into a domain error so
		// delivery handlers can map upstream 4xx responses (e.g. 400 for an
		// empty keyword) to a client-facing status.
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg}
		}
		return nil, err
	}
	return toDomainSearch(resp.Data), nil
}

func toDomainSearch(d stockbit.SearchData) *domain.SearchResult {
	out := &domain.SearchResult{
		Chat:       make([]domain.SearchChat, 0, len(d.Chat)),
		Company:    make([]domain.SearchCompany, 0, len(d.Company)),
		Insider:    make([]domain.SearchInsider, 0, len(d.Insider)),
		People:     make([]domain.SearchPerson, 0, len(d.People)),
		Sector:     make([]domain.SearchSector, 0, len(d.Sector)),
		Industries: make([]domain.SearchIndustry, 0, len(d.Industries)),
		Pagination: domain.SearchPagination{
			HasMoreCompanies: d.Pagination.HasMoreCompanies,
			HasMoreInsiders:  d.Pagination.HasMoreInsiders,
			HasMoreUsers:     d.Pagination.HasMoreUsers,
		},
	}
	for _, c := range d.Company {
		out.Company = append(out.Company, domain.SearchCompany{
			ID:          c.ID,
			Name:        c.Name,
			Country:     c.Country,
			Desc:        c.Desc,
			Exchange:    c.Exchange,
			IsFollowing: c.IsFollowing,
			Img:         c.Img,
			IsVerified:  c.IsVerified,
			Other:       c.Other,
			Status:      c.Status,
			Symbol2:     c.Symbol2,
			Symbol3:     c.Symbol3,
			TotalFollow: c.TotalFollow,
			IsTradeable: c.IsTradeable,
			Type:        c.Type,
			URL:         c.URL,
			IconURL:     c.IconURL,
		})
	}
	for _, it := range d.Insider {
		out.Insider = append(out.Insider, domain.SearchInsider{
			ID:          it.ID,
			IsFollowing: it.IsFollowing,
			Label:       it.Label,
			IsVerified:  it.IsVerified,
			Permalink:   it.Permalink,
			Tradeable:   it.Tradeable,
			Type:        it.Type,
		})
	}
	for _, c := range d.Chat {
		out.Chat = append(out.Chat, domain.SearchChat{
			ID:          c.ID,
			Label:       c.Label,
			ChatType:    c.ChatType,
			IsFollowing: c.IsFollowing,
			IsVerified:  c.IsVerified,
			IconURL:     c.IconURL,
		})
	}
	for _, p := range d.People {
		out.People = append(out.People, domain.SearchPerson{
			ID:          p.ID,
			Label:       p.Label,
			IsFollowing: p.IsFollowing,
			IsVerified:  p.IsVerified,
			IconURL:     p.IconURL,
			Permalink:   p.Permalink,
		})
	}
	for _, s := range d.Sector {
		out.Sector = append(out.Sector, domain.SearchSector{
			ID:       s.ID,
			Name:     s.Name,
			IconURL:  s.IconURL,
			IsFollow: s.IsFollow,
		})
	}
	for _, i := range d.Industries {
		out.Industries = append(out.Industries, domain.SearchIndustry{
			ID:       i.ID,
			Name:     i.Name,
			IconURL:  i.IconURL,
			IsFollow: i.IsFollow,
		})
	}
	return out
}

var _ repository.SearchRepository = (*SearchRepository)(nil)
