package stockapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

// This file implements repository.CompanyProfileSyncClient on *Client: it
// fetches one company's profile cluster from the apps/api endpoints
// documented in docs/api.md:
//   - GET /api/v1/company/{symbol}/profile        (header + people + holdings)
//   - GET /api/v1/company/{symbol}/subsidiaries   (rich subsidiary rows)
//
// Field names and value formats below mirror the upstream payload exactly.

// --- Profile endpoint payload ------------------------------------------------

type upstreamProfileEnvelope struct {
	Success bool             `json:"success"`
	Data    *upstreamProfile `json:"data"`
}

type upstreamProfile struct {
	Background         string                      `json:"background"`
	History            *upstreamHistory            `json:"history"`
	KeyExec            *upstreamKeyExec            `json:"key_executive"`
	Address            []upstreamAddress           `json:"address"`
	ShareholderNumbers []upstreamShareholderNumber `json:"shareholder_numbers"`
	// Holding groups (one array per upstream group).
	Shareholder                     []upstreamHolding `json:"shareholder"`
	ShareholderOnePercent           []upstreamHolding `json:"shareholder_one_percent"`
	ShareholderDirectorCommissioner []upstreamHolding `json:"shareholder_director_commissioner"`
	Beneficiary                     []upstreamHolding `json:"beneficiary"`
	// Subsidiary here is the lean per-profile list; the dedicated
	// /subsidiaries endpoint is authoritative for company_subsidiaries and
	// is fetched separately, so this field is intentionally ignored.
}

type upstreamHistory struct {
	Board     string `json:"board"`
	Date      string `json:"date"`
	Price     string `json:"price"`
	Amount    string `json:"amount"`
	Shares    string `json:"shares"`
	FreeFloat string `json:"free_float"`
	Registrar string `json:"registrar"`
}

type upstreamKeyExec struct {
	Commissioner            []upstreamExec `json:"commissioner"`
	Director                []upstreamExec `json:"director"`
	IndependentCommissioner []upstreamExec `json:"independent_commissioner"`
}

// upstreamExec is one executive {id, key, value}: key is the role label
// ("Commissioner", "Director", "Commissioner (Independent)"), value the name.
type upstreamExec struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type upstreamHolding struct {
	Name       string   `json:"name"`
	Percentage string   `json:"percentage"`
	Value      string   `json:"value"`
	Badges     []string `json:"badges"`
}

type upstreamShareholderNumber struct {
	ShareholderDate string `json:"shareholder_date"`
	TotalShare      string `json:"total_share"`
	Change          *int64 `json:"change"`
	ChangeFormatted string `json:"change_formatted"`
}

type upstreamAddress struct {
	Office  string   `json:"office"`
	Phone   string   `json:"phone"`
	Fax     string   `json:"fax"`
	Website string   `json:"website"`
	Npwp    string   `json:"npwp"`
	Emails  []string `json:"email"`
}

// --- Subsidiaries endpoint payload ------------------------------------------

type upstreamSubsidiariesEnvelope struct {
	Success bool                  `json:"success"`
	Data    *upstreamSubsidiaries `json:"data"`
}

type upstreamSubsidiaries struct {
	Subsidiaries []upstreamSubsidiary `json:"subsidiaries"`
}

type upstreamSubsidiary struct {
	CompanyName       string `json:"company_name"`
	BusinessType      string `json:"business_type"`
	Location          string `json:"location"`
	CommercialYear    string `json:"commercial_year"`
	TotalAssets       string `json:"total_assets"`
	Percentage        string `json:"percentage"`
	OperationalStatus string `json:"operational_status"`
	Period            string `json:"period"`
}

// FetchCompanyProfile implements repository.CompanyProfileSyncClient. It
// fetches the profile cluster for one symbol from apps/api and maps it into
// the domain aggregate. Any upstream failure aborts the whole fetch (the
// caller keeps whatever it already had — no partial replace).
func (c *Client) FetchCompanyProfile(ctx context.Context, symbol string) (domain.CompanyProfile, error) {
	profile, err := c.fetchProfile(ctx, symbol)
	if err != nil {
		return domain.CompanyProfile{}, err
	}
	subsidiaries, err := c.fetchSubsidiaries(ctx, symbol)
	if err != nil {
		return domain.CompanyProfile{}, err
	}
	profile.Subsidiaries = subsidiaries
	return profile, nil
}

func (c *Client) fetchProfile(ctx context.Context, symbol string) (domain.CompanyProfile, error) {
	u, err := profileURL(c.baseURL, "/api/v1/company/"+url.PathEscape(symbol)+"/profile")
	if err != nil {
		return domain.CompanyProfile{}, err
	}
	body, err := c.doGet(ctx, u)
	if err != nil {
		return domain.CompanyProfile{}, fmt.Errorf("stockapi: fetch profile %s: %w", symbol, err)
	}

	var env upstreamProfileEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return domain.CompanyProfile{}, fmt.Errorf("stockapi: decode profile %s: %w", symbol, err)
	}
	if env.Data == nil {
		return domain.CompanyProfile{}, fmt.Errorf("stockapi: profile %s: empty upstream data", symbol)
	}

	return mapProfile(symbol, env.Data), nil
}

func (c *Client) fetchSubsidiaries(ctx context.Context, symbol string) ([]domain.CompanySubsidiary, error) {
	u, err := profileURL(c.baseURL, "/api/v1/company/"+url.PathEscape(symbol)+"/subsidiaries")
	if err != nil {
		return nil, err
	}
	body, err := c.doGet(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("stockapi: fetch subsidiaries %s: %w", symbol, err)
	}

	var env upstreamSubsidiariesEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("stockapi: decode subsidiaries %s: %w", symbol, err)
	}
	if env.Data == nil {
		return nil, fmt.Errorf("stockapi: subsidiaries %s: empty upstream data", symbol)
	}

	out := make([]domain.CompanySubsidiary, 0, len(env.Data.Subsidiaries))
	for i, s := range env.Data.Subsidiaries {
		out = append(out, domain.CompanySubsidiary{
			Name:              s.CompanyName,
			BusinessType:      nullable(s.BusinessType),
			Location:          nullable(s.Location),
			CommercialYear:    nullable(s.CommercialYear),
			TotalAssets:       parseDecimal(s.TotalAssets),
			TotalAssetsRaw:    nullable(s.TotalAssets),
			Percentage:        parseDecimal(s.Percentage),
			PercentageRaw:     nullable(s.Percentage),
			OperationalStatus: nullable(s.OperationalStatus),
			Period:            nullable(s.Period),
			Position:          i,
		})
	}
	return out, nil
}

func mapProfile(symbol string, p *upstreamProfile) domain.CompanyProfile {
	out := domain.CompanyProfile{Symbol: symbol, Background: nullable(p.Background)}
	if h := p.History; h != nil {
		out.Board = nullable(h.Board)
		out.ListingDate = nullable(h.Date)
		out.ListingPrice = nullable(h.Price)
		out.IpoAmount = nullable(h.Amount)
		out.ListedShares = nullable(h.Shares)
		out.FreeFloat = nullable(h.FreeFloat)
		out.Registrar = nullable(h.Registrar)
	}
	if ke := p.KeyExec; ke != nil {
		pos := 0
		for _, group := range []struct {
			kind string
			list []upstreamExec
		}{
			{"commissioner", ke.Commissioner},
			{"director", ke.Director},
			{"independent_commissioner", ke.IndependentCommissioner},
		} {
			for _, e := range group.list {
				out.Executives = append(out.Executives, domain.CompanyExecutive{
					Kind:       group.kind,
					Name:       e.Value,
					Role:       nullable(e.Key),
					ExternalID: nullable(e.ID),
					Position:   pos,
				})
				pos++
			}
		}
	}

	pos := 0
	for _, group := range []struct {
		holderGroup string
		list        []upstreamHolding
	}{
		{"shareholder", p.Shareholder},
		{"one_percent", p.ShareholderOnePercent},
		{"director_commissioner", p.ShareholderDirectorCommissioner},
		{"beneficiary", p.Beneficiary},
	} {
		for _, h := range group.list {
			out.Holdings = append(out.Holdings, domain.CompanyHolding{
				HolderGroup:   group.holderGroup,
				Name:          h.Name,
				Percentage:    parseDecimal(h.Percentage),
				PercentageRaw: nullable(h.Percentage),
				AmountRaw:     nullable(h.Value),
				Badges:        h.Badges,
				Position:      pos,
			})
			pos++
		}
	}

	for _, n := range p.ShareholderNumbers {
		out.ShareholderNumbers = append(out.ShareholderNumbers, domain.CompanyShareholderNumber{
			ShareholderDate: n.ShareholderDate,
			TotalShare:      nullable(n.TotalShare),
			Change:          n.Change,
			ChangeFormatted: nullable(n.ChangeFormatted),
		})
	}

	for i, a := range p.Address {
		out.Addresses = append(out.Addresses, domain.CompanyAddress{
			Office:   nullable(a.Office),
			Phone:    nullable(a.Phone),
			Fax:      nullable(a.Fax),
			Website:  nullable(a.Website),
			Npwp:     nullable(a.Npwp),
			Emails:   a.Emails,
			Position: i,
		})
	}
	return out
}

func profileURL(base, path string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("stockapi: parse base url: %w", err)
	}
	u.Path = path
	return u.String(), nil
}

// doGet performs a GET and returns the raw body, failing on non-2xx.
func (c *Client) doGet(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("stockapi: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stockapi: do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("stockapi: read response: %w", err)
	}
	return body, nil
}

// parseDecimal converts upstream percentage/number strings such as
// "54.942%", "100.00" or "20,818,005" into a float. Unparseable or empty
// values yield nil rather than an error, mirroring the nullable DB columns.
func parseDecimal(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "%", "")
	s = strings.ReplaceAll(s, ",", "")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &f
}
