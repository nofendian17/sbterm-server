package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// CompanyProfileRepository fetches a company profile from the Stockbit API.
type CompanyProfileRepository struct {
	client *stockbit.Client
}

func NewCompanyProfileRepository(client *stockbit.Client) *CompanyProfileRepository {
	return &CompanyProfileRepository{client: client}
}

func (r *CompanyProfileRepository) GetProfile(ctx context.Context, symbol string) (*domain.CompanyProfile, error) {
	resp, err := r.client.GetProfile(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return toProfileDomain(resp.Data), nil
}

var _ repository.CompanyProfileRepository = (*CompanyProfileRepository)(nil)

func toProfileDomain(p stockbit.CompanyProfile) *domain.CompanyProfile {
	out := &domain.CompanyProfile{
		Background:                      p.Background,
		History:                         nil,
		Address:                         make([]domain.ProfileAddress, 0, len(p.Address)),
		Subsidiary:                      make([]domain.ProfileSubsidiary, 0, len(p.Subsidiary)),
		Beneficiary:                     make([]domain.ProfileBeneficiary, 0, len(p.Beneficiary)),
		Shareholder:                     mapShareholders(p.Shareholder),
		ShareholderDirectorCommissioner: mapShareholders(p.ShareholderDirectorCommissioner),
		ShareholderNumbers:              mapShareholderNumbers(p.ShareholderNumbers),
	}
	if p.ShareholderOnePercent != nil {
		out.ShareholderOnePercent = mapShareholders(p.ShareholderOnePercent.Shareholder)
	}
	if p.History != nil {
		out.History = &domain.ProfileHistory{
			Amount:       p.History.Amount,
			Board:        p.History.Board,
			Date:         p.History.Date,
			Price:        p.History.Price,
			Registrar:    p.History.Registrar,
			Shares:       p.History.Shares,
			Underwriters: p.History.Underwriters,
			FreeFloat:    p.History.FreeFloat,
		}
	}
	if p.KeyExecutive != nil {
		out.KeyExecutive = &domain.ProfileKeyExecutive{
			Commissioner:            mapExecutives(p.KeyExecutive.Commissioner),
			Director:                mapExecutives(p.KeyExecutive.Director),
			IndependentCommissioner: mapExecutives(p.KeyExecutive.IndependentCommissioner),
		}
	}
	for _, a := range p.Address {
		out.Address = append(out.Address, domain.ProfileAddress{
			Office:  a.Office,
			Phone:   a.Phone,
			Fax:     a.Fax,
			Email:   a.Email,
			Website: a.Website,
			NPWP:    a.NPWP,
		})
	}
	for _, s := range p.Subsidiary {
		out.Subsidiary = append(out.Subsidiary, domain.ProfileSubsidiary{
			Company:    s.Company,
			Percentage: s.Percentage,
			Types:      s.Types,
			Value:      s.Value,
		})
	}
	for _, b := range p.Beneficiary {
		out.Beneficiary = append(out.Beneficiary, domain.ProfileBeneficiary{Name: b.Name})
	}
	return out
}

func mapExecutives(in []stockbit.ProfileExecutive) []domain.ProfileExecutive {
	out := make([]domain.ProfileExecutive, 0, len(in))
	for _, e := range in {
		out = append(out, domain.ProfileExecutive{ID: e.ID, Key: e.Key, Value: e.Value})
	}
	return out
}

func mapShareholders(in []stockbit.ProfileShareholder) []domain.ProfileShareholder {
	out := make([]domain.ProfileShareholder, 0, len(in))
	for _, s := range in {
		out = append(out, domain.ProfileShareholder{
			ID:             s.ID,
			Percentage:     s.Percentage,
			Name:           s.Name,
			Value:          s.Value,
			Badges:         s.Badges,
			Type:           s.Type,
			Location:       s.Location,
			Nationality:    s.Nationality,
			Domicile:       s.Domicile,
			Scripless:      s.Scripless,
			Scrip:          s.Scrip,
			ValueFormatted: s.ValueFormatted,
			Classification: s.Classification,
		})
	}
	return out
}

func mapShareholderNumbers(in []stockbit.ProfileShareholderNumber) []domain.ProfileShareholderNumber {
	out := make([]domain.ProfileShareholderNumber, 0, len(in))
	for _, n := range in {
		out = append(out, domain.ProfileShareholderNumber{
			ShareholderDate: n.ShareholderDate,
			TotalShare:      n.TotalShare,
			Change:          n.Change,
			ChangeFormatted: n.ChangeFormatted,
		})
	}
	return out
}
