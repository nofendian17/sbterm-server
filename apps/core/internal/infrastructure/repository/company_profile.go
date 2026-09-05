package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	contract "github.com/nofendian17/sbterm/apps/core/internal/repository"
)

// CompanyProfileRepository is the pgx implementation of
// contract.CompanyProfileRepository. Reads run on the pool querier; Save
// replaces the whole cluster inside a transaction via the TxManager.
type CompanyProfileRepository struct {
	q   contract.Querier
	txm contract.TxManager
}

// NewCompanyProfileRepository builds a CompanyProfileRepository.
func NewCompanyProfileRepository(q contract.Querier, txm contract.TxManager) *CompanyProfileRepository {
	return &CompanyProfileRepository{q: q, txm: txm}
}

// GetBySymbol returns the full profile aggregate, or
// domain.ErrCompanyProfileNotFound when the stock has no profile row.
func (r *CompanyProfileRepository) GetBySymbol(ctx context.Context, symbol string) (domain.CompanyProfile, error) {
	p := domain.CompanyProfile{Symbol: symbol}

	err := r.q.QueryRow(ctx,
		`SELECT background, board, listing_date, listing_price, ipo_amount,
		        listed_shares, free_float, registrar
		 FROM company_profiles WHERE symbol = $1 AND deleted_at IS NULL`, symbol,
	).Scan(
		&p.Background, &p.Board, &p.ListingDate, &p.ListingPrice, &p.IpoAmount,
		&p.ListedShares, &p.FreeFloat, &p.Registrar,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.CompanyProfile{}, fmt.Errorf("company profile get: %w", domain.ErrCompanyProfileNotFound)
		}
		return domain.CompanyProfile{}, fmt.Errorf("company profile get: %w", err)
	}

	if p.Executives, err = r.executives(ctx, symbol); err != nil {
		return domain.CompanyProfile{}, err
	}
	if p.Holdings, err = r.holdings(ctx, symbol); err != nil {
		return domain.CompanyProfile{}, err
	}
	if p.ShareholderNumbers, err = r.shareholderNumbers(ctx, symbol); err != nil {
		return domain.CompanyProfile{}, err
	}
	if p.Subsidiaries, err = r.subsidiaries(ctx, symbol); err != nil {
		return domain.CompanyProfile{}, err
	}
	if p.Addresses, err = r.addresses(ctx, symbol); err != nil {
		return domain.CompanyProfile{}, err
	}
	return p, nil
}

func (r *CompanyProfileRepository) executives(ctx context.Context, symbol string) ([]domain.CompanyExecutive, error) {
	rows, err := r.q.Query(ctx,
		`SELECT kind, name, role, external_id, position FROM company_executives
		 WHERE symbol = $1 AND deleted_at IS NULL ORDER BY position, id`, symbol)
	if err != nil {
		return nil, fmt.Errorf("company profile executives: %w", err)
	}
	defer rows.Close()
	out := []domain.CompanyExecutive{}
	for rows.Next() {
		var e domain.CompanyExecutive
		if err := rows.Scan(&e.Kind, &e.Name, &e.Role, &e.ExternalID, &e.Position); err != nil {
			return nil, fmt.Errorf("company profile executives scan: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *CompanyProfileRepository) holdings(ctx context.Context, symbol string) ([]domain.CompanyHolding, error) {
	rows, err := r.q.Query(ctx,
		`SELECT holder_group, name, percentage, percentage_raw, amount_raw, badges, position
		 FROM company_holdings WHERE symbol = $1 AND deleted_at IS NULL ORDER BY position, id`, symbol)
	if err != nil {
		return nil, fmt.Errorf("company profile holdings: %w", err)
	}
	defer rows.Close()
	out := []domain.CompanyHolding{}
	for rows.Next() {
		var h domain.CompanyHolding
		if err := rows.Scan(&h.HolderGroup, &h.Name, &h.Percentage, &h.PercentageRaw, &h.AmountRaw, &h.Badges, &h.Position); err != nil {
			return nil, fmt.Errorf("company profile holdings scan: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *CompanyProfileRepository) shareholderNumbers(ctx context.Context, symbol string) ([]domain.CompanyShareholderNumber, error) {
	rows, err := r.q.Query(ctx,
		`SELECT shareholder_date, total_share, change, change_formatted
		 FROM company_shareholder_numbers WHERE symbol = $1 AND deleted_at IS NULL ORDER BY id`, symbol)
	if err != nil {
		return nil, fmt.Errorf("company profile shareholder numbers: %w", err)
	}
	defer rows.Close()
	out := []domain.CompanyShareholderNumber{}
	for rows.Next() {
		var n domain.CompanyShareholderNumber
		if err := rows.Scan(&n.ShareholderDate, &n.TotalShare, &n.Change, &n.ChangeFormatted); err != nil {
			return nil, fmt.Errorf("company profile shareholder numbers scan: %w", err)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *CompanyProfileRepository) subsidiaries(ctx context.Context, symbol string) ([]domain.CompanySubsidiary, error) {
	rows, err := r.q.Query(ctx,
		`SELECT name, business_type, location, commercial_year, total_assets, total_assets_raw,
		        percentage, percentage_raw, operational_status, period, position
		 FROM company_subsidiaries WHERE symbol = $1 AND deleted_at IS NULL ORDER BY position, id`, symbol)
	if err != nil {
		return nil, fmt.Errorf("company profile subsidiaries: %w", err)
	}
	defer rows.Close()
	out := []domain.CompanySubsidiary{}
	for rows.Next() {
		var s domain.CompanySubsidiary
		if err := rows.Scan(
			&s.Name, &s.BusinessType, &s.Location, &s.CommercialYear,
			&s.TotalAssets, &s.TotalAssetsRaw, &s.Percentage, &s.PercentageRaw,
			&s.OperationalStatus, &s.Period, &s.Position,
		); err != nil {
			return nil, fmt.Errorf("company profile subsidiaries scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *CompanyProfileRepository) addresses(ctx context.Context, symbol string) ([]domain.CompanyAddress, error) {
	rows, err := r.q.Query(ctx,
		`SELECT office, phone, fax, website, npwp, emails, position
		 FROM company_addresses WHERE symbol = $1 AND deleted_at IS NULL ORDER BY position, id`, symbol)
	if err != nil {
		return nil, fmt.Errorf("company profile addresses: %w", err)
	}
	defer rows.Close()
	out := []domain.CompanyAddress{}
	for rows.Next() {
		var a domain.CompanyAddress
		if err := rows.Scan(&a.Office, &a.Phone, &a.Fax, &a.Website, &a.Npwp, &a.Emails, &a.Position); err != nil {
			return nil, fmt.Errorf("company profile addresses scan: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Save replaces the whole profile cluster for the symbol atomically. The
// header is upserted (reactivating a soft-deleted row) and every child table
// is cleared and repopulated from the aggregate.
func (r *CompanyProfileRepository) Save(ctx context.Context, p domain.CompanyProfile) error {
	err := r.txm.WithTx(ctx, func(tx contract.Querier) error {
		if err := upsertProfileHeader(ctx, tx, p); err != nil {
			return err
		}
		for _, del := range []string{
			"company_executives",
			"company_holdings",
			"company_shareholder_numbers",
			"company_subsidiaries",
			"company_addresses",
		} {
			if _, err := tx.Exec(ctx,
				`DELETE FROM `+del+` WHERE symbol = $1`, p.Symbol); err != nil {
				return fmt.Errorf("company profile clear %s: %w", del, err)
			}
		}
		if err := insertExecutives(ctx, tx, p.Symbol, p.Executives); err != nil {
			return err
		}
		if err := insertHoldings(ctx, tx, p.Symbol, p.Holdings); err != nil {
			return err
		}
		if err := insertShareholderNumbers(ctx, tx, p.Symbol, p.ShareholderNumbers); err != nil {
			return err
		}
		if err := insertSubsidiaries(ctx, tx, p.Symbol, p.Subsidiaries); err != nil {
			return err
		}
		return insertAddresses(ctx, tx, p.Symbol, p.Addresses)
	})
	if err != nil {
		// Saving a profile for a symbol that is not in the stocks master
		// table trips the header FK (23503).
		if isPgErrorCode(err, foreignKeyViolationCode) {
			return fmt.Errorf("company profile save: %w", domain.ErrStockNotFound)
		}
		return fmt.Errorf("company profile save: %w", err)
	}
	return nil
}

func upsertProfileHeader(ctx context.Context, tx contract.Querier, p domain.CompanyProfile) error {
	const q = `
		INSERT INTO company_profiles (symbol, background, board, listing_date,
			listing_price, ipo_amount, listed_shares, free_float, registrar)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (symbol) DO UPDATE
		SET background = EXCLUDED.background,
		    board = EXCLUDED.board,
		    listing_date = EXCLUDED.listing_date,
		    listing_price = EXCLUDED.listing_price,
		    ipo_amount = EXCLUDED.ipo_amount,
		    listed_shares = EXCLUDED.listed_shares,
		    free_float = EXCLUDED.free_float,
		    registrar = EXCLUDED.registrar,
		    deleted_at = NULL,
		    updated_at = now()
	`
	if _, err := tx.Exec(ctx, q,
		p.Symbol, p.Background, p.Board, p.ListingDate, p.ListingPrice,
		p.IpoAmount, p.ListedShares, p.FreeFloat, p.Registrar,
	); err != nil {
		return fmt.Errorf("company profile upsert header: %w", err)
	}
	return nil
}

func insertExecutives(ctx context.Context, tx contract.Querier, symbol string, items []domain.CompanyExecutive) error {
	for i, e := range items {
		const q = `INSERT INTO company_executives (symbol, kind, name, role, external_id, position)
			VALUES ($1, $2, $3, $4, $5, $6)`
		if _, err := tx.Exec(ctx, q, symbol, e.Kind, e.Name, e.Role, e.ExternalID, orderedPosition(e.Position, i)); err != nil {
			return fmt.Errorf("company profile insert executive: %w", err)
		}
	}
	return nil
}

func insertHoldings(ctx context.Context, tx contract.Querier, symbol string, items []domain.CompanyHolding) error {
	for i, h := range items {
		const q = `INSERT INTO company_holdings (symbol, holder_group, name, percentage,
				percentage_raw, amount_raw, badges, position)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
		if _, err := tx.Exec(ctx, q, symbol, h.HolderGroup, h.Name, h.Percentage,
			h.PercentageRaw, h.AmountRaw, arrayOrEmpty(h.Badges), orderedPosition(h.Position, i)); err != nil {
			return fmt.Errorf("company profile insert holding: %w", err)
		}
	}
	return nil
}

func insertShareholderNumbers(ctx context.Context, tx contract.Querier, symbol string, items []domain.CompanyShareholderNumber) error {
	for _, n := range items {
		const q = `INSERT INTO company_shareholder_numbers (symbol, shareholder_date, total_share, change, change_formatted)
			VALUES ($1, $2, $3, $4, $5)`
		if _, err := tx.Exec(ctx, q, symbol, n.ShareholderDate, n.TotalShare, n.Change, n.ChangeFormatted); err != nil {
			return fmt.Errorf("company profile insert shareholder number: %w", err)
		}
	}
	return nil
}

func insertSubsidiaries(ctx context.Context, tx contract.Querier, symbol string, items []domain.CompanySubsidiary) error {
	for i, s := range items {
		const q = `INSERT INTO company_subsidiaries (symbol, name, business_type, location,
				commercial_year, total_assets, total_assets_raw, percentage, percentage_raw,
				operational_status, period, position)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
		if _, err := tx.Exec(ctx, q, symbol, s.Name, s.BusinessType, s.Location,
			s.CommercialYear, s.TotalAssets, s.TotalAssetsRaw, s.Percentage, s.PercentageRaw,
			s.OperationalStatus, s.Period, orderedPosition(s.Position, i)); err != nil {
			return fmt.Errorf("company profile insert subsidiary: %w", err)
		}
	}
	return nil
}

func insertAddresses(ctx context.Context, tx contract.Querier, symbol string, items []domain.CompanyAddress) error {
	for i, a := range items {
		const q = `INSERT INTO company_addresses (symbol, office, phone, fax, website, npwp, emails, position)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
		if _, err := tx.Exec(ctx, q, symbol, a.Office, a.Phone, a.Fax, a.Website, a.Npwp, arrayOrEmpty(a.Emails), orderedPosition(a.Position, i)); err != nil {
			return fmt.Errorf("company profile insert address: %w", err)
		}
	}
	return nil
}

// orderedPosition uses the caller-provided position when > 0, otherwise the
// slice index, so list order survives even when clients omit positions.
func orderedPosition(pos, index int) int {
	if pos > 0 {
		return pos
	}
	return index
}

// arrayOrEmpty maps a nil slice to an empty one so pgx encodes '{}' (not
// NULL) into the NOT NULL TEXT[] columns.
func arrayOrEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
