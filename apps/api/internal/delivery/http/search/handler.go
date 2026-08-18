package search

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

const (
	defaultPage = 1
	defaultType = "company"
)

type SearchHandler struct {
	uc usecase.SearchUsecase
	v  validator.Validator
}

func NewSearchHandler(uc usecase.SearchUsecase, v validator.Validator) *SearchHandler {
	return &SearchHandler{uc: uc, v: v}
}

type searchRequest struct {
	Keyword string `json:"keyword" validate:"required"`
	Page    int    `json:"page" validate:"min=1"`
	Type    string `json:"type" validate:"omitempty,oneof=company insider people sector industries chat all"`
}

type searchResponse struct {
	Chat       []searchChatResp     `json:"chat"`
	Company    []searchCompanyResp  `json:"company"`
	Insider    []searchInsiderResp  `json:"insider"`
	People     []searchPeopleResp   `json:"people"`
	Sector     []searchSectorResp   `json:"sector"`
	Industries []searchIndustryResp `json:"industries"`
	Pagination searchPaginationResp `json:"pagination"`
}

type searchCompanyResp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Country     string `json:"country"`
	Desc        string `json:"desc"`
	Exchange    string `json:"exchange"`
	IsFollowing bool   `json:"is_following"`
	Img         string `json:"img"`
	IsVerified  bool   `json:"is_verified"`
	Other       string `json:"other"`
	Status      string `json:"status"`
	Symbol2     string `json:"symbol_2"`
	Symbol3     string `json:"symbol_3"`
	TotalFollow int    `json:"total_followers"`
	IsTradeable bool   `json:"is_tradeable"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	IconURL     string `json:"icon_url"`
}

type searchInsiderResp struct {
	ID          string `json:"id"`
	IsFollowing bool   `json:"is_following"`
	Label       string `json:"label"`
	IsVerified  bool   `json:"is_verified"`
	Permalink   string `json:"permalink"`
	Tradeable   int    `json:"tradeable"`
	Type        string `json:"type"`
}

type searchChatResp struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	ChatType    string `json:"chat_type"`
	IsFollowing bool   `json:"is_following"`
	IsVerified  bool   `json:"is_verified"`
	IconURL     string `json:"icon_url"`
}

type searchPeopleResp struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	IsFollowing bool   `json:"is_following"`
	IsVerified  bool   `json:"is_verified"`
	IconURL     string `json:"icon_url"`
	Permalink   string `json:"permalink"`
}

type searchSectorResp struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IconURL  string `json:"icon_url"`
	IsFollow bool   `json:"is_follow"`
}

type searchIndustryResp struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IconURL  string `json:"icon_url"`
	IsFollow bool   `json:"is_follow"`
}

type searchPaginationResp struct {
	HasMoreCompanies bool `json:"has_more_companies"`
	HasMoreInsiders  bool `json:"has_more_insiders"`
	HasMoreUsers     bool `json:"has_more_users"`
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	page, err := parseIntQuery(r.URL.Query().Get("page"), defaultPage)
	if err != nil {
		response.ValidationError(w, "validation failed", map[string]string{"page": "must be a valid integer"})
		return
	}
	req := searchRequest{
		Keyword: r.URL.Query().Get("keyword"),
		Page:    page,
		Type:    r.URL.Query().Get("type"),
	}
	if req.Type == "" {
		req.Type = defaultType
	}
	if err := h.v.Validate(req); err != nil {
		if verr, ok := validator.AsValidationError(err); ok {
			response.ValidationError(w, "validation failed", verr.Fields)
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to validate search params")
		return
	}

	result, err := h.uc.GetSearch(r.Context(), req.Keyword, req.Page, req.Type)
	if err != nil {
		var upErr *domain.UpstreamError
		if errors.As(err, &upErr) && upErr.Status == http.StatusBadRequest {
			response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "invalid search parameters")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to search")
		return
	}
	response.OK(w, toResponse(result))
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

func toResponse(d *domain.SearchResult) searchResponse {
	out := searchResponse{
		Chat:       make([]searchChatResp, 0, len(d.Chat)),
		Company:    make([]searchCompanyResp, 0, len(d.Company)),
		Insider:    make([]searchInsiderResp, 0, len(d.Insider)),
		People:     make([]searchPeopleResp, 0, len(d.People)),
		Sector:     make([]searchSectorResp, 0, len(d.Sector)),
		Industries: make([]searchIndustryResp, 0, len(d.Industries)),
		Pagination: searchPaginationResp{
			HasMoreCompanies: d.Pagination.HasMoreCompanies,
			HasMoreInsiders:  d.Pagination.HasMoreInsiders,
			HasMoreUsers:     d.Pagination.HasMoreUsers,
		},
	}
	for _, c := range d.Company {
		out.Company = append(out.Company, searchCompanyResp{
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
		out.Insider = append(out.Insider, searchInsiderResp{
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
		out.Chat = append(out.Chat, searchChatResp{
			ID:          c.ID,
			Label:       c.Label,
			ChatType:    c.ChatType,
			IsFollowing: c.IsFollowing,
			IsVerified:  c.IsVerified,
			IconURL:     c.IconURL,
		})
	}
	for _, p := range d.People {
		out.People = append(out.People, searchPeopleResp{
			ID:          p.ID,
			Label:       p.Label,
			IsFollowing: p.IsFollowing,
			IsVerified:  p.IsVerified,
			IconURL:     p.IconURL,
			Permalink:   p.Permalink,
		})
	}
	for _, s := range d.Sector {
		out.Sector = append(out.Sector, searchSectorResp{
			ID:       s.ID,
			Name:     s.Name,
			IconURL:  s.IconURL,
			IsFollow: s.IsFollow,
		})
	}
	for _, i := range d.Industries {
		out.Industries = append(out.Industries, searchIndustryResp{
			ID:       i.ID,
			Name:     i.Name,
			IconURL:  i.IconURL,
			IsFollow: i.IsFollow,
		})
	}
	return out
}
