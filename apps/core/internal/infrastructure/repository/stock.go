// Package repository implements the repository contracts using PostgreSQL/Redis backends.

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	contract "github.com/nofendian17/sbterm/apps/core/internal/repository"
)

// StockRepository is the pgx implementation of contract.StockRepository. It
// runs every query through a contract.Querier so the same code works outside
// (pool) and inside (tx) a transaction.
type StockRepository struct {
	q contract.Querier
}

// NewStockRepository builds a StockRepository backed by the given Querier.
func NewStockRepository(q contract.Querier) *StockRepository {
	return &StockRepository{q: q}
}

// stockSelectCols is the shared projection for stock reads. The sector LEFT
// JOIN is always present so responses can carry the sector name.
const stockSelectCols = `s.symbol, s.name, s.sector_id, sec.name, s.exchange, s.icon_url, s.is_active, s.synced_at, s.created_at, s.updated_at`

const stockFromJoin = `FROM stocks s
	 LEFT JOIN sectors sec ON sec.id = s.sector_id AND sec.deleted_at IS NULL`

// scanStockRow scans one row produced by stockSelectCols into domain.Stock.
// Nullable text columns are read through sql.NullString (works with both real
// pgx and pgxmock, which cannot scan into **T destinations).
func scanStockRow(row interface{ Scan(dest ...any) error }) (domain.Stock, error) {
	var (
		s        domain.Stock
		sectorID sql.NullString
		sectorNm sql.NullString
		exchange sql.NullString
		iconURL  sql.NullString
	)
	if err := row.Scan(
		&s.Symbol,
		&s.Name,
		&sectorID,
		&sectorNm,
		&exchange,
		&iconURL,
		&s.IsActive,
		&s.SyncedAt,
		&s.CreatedAt,
		&s.UpdatedAt,
	); err != nil {
		return domain.Stock{}, err
	}
	s.Exchange = nullStringPtr(exchange)
	s.IconURL = nullStringPtr(iconURL)
	if sectorID.Valid {
		id := sectorID.String
		s.SectorID = &id
		s.Sector = &domain.Sector{ID: id, Name: sectorNm.String}
	}
	return s, nil
}

// GetBySymbol returns the non-deleted stock with the given symbol.
func (r *StockRepository) GetBySymbol(ctx context.Context, symbol string) (domain.Stock, error) {
	q := `SELECT ` + stockSelectCols + `
		` + stockFromJoin + `
		WHERE s.symbol = $1 AND s.deleted_at IS NULL`
	s, err := scanStockRow(r.q.QueryRow(ctx, q, symbol))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Stock{}, fmt.Errorf("stock get: %w", domain.ErrStockNotFound)
		}
		return domain.Stock{}, fmt.Errorf("stock get: %w", err)
	}
	return s, nil
}

// List returns a page of stocks matching the filter.
func (r *StockRepository) List(ctx context.Context, f domain.StockFilter) ([]domain.Stock, int, error) {
	page, limit := domain.NormalizePagination(f.Page, f.Limit)

	var (
		where []string
		args  []any
	)
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where = append(where, fmt.Sprintf("(s.symbol ILIKE $%d OR s.name ILIKE $%d)", len(args), len(args)))
	}
	if f.Sector != "" {
		args = append(args, f.Sector)
		where = append(where, fmt.Sprintf("sec.name = $%d", len(args)))
	}
	if f.IsActive != nil {
		args = append(args, *f.IsActive)
		where = append(where, fmt.Sprintf("s.is_active = $%d", len(args)))
	}
	whereClause := "WHERE s.deleted_at IS NULL"
	if len(where) > 0 {
		whereClause += " AND " + strings.Join(where, " AND ")
	}

	var total int
	if err := r.q.QueryRow(ctx,
		`SELECT COUNT(*) `+stockFromJoin+` `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("stock list count: %w", err)
	}

	pageArgs := append(append([]any{}, args...), limit, (page-1)*limit)
	rows, err := r.q.Query(ctx,
		`SELECT `+stockSelectCols+` `+stockFromJoin+` `+whereClause+
			fmt.Sprintf(` ORDER BY s.symbol LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2),
		pageArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("stock list page: %w", err)
	}
	defer rows.Close()

	out := []domain.Stock{}
	for rows.Next() {
		s, err := scanStockRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("stock list scan: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("stock list rows: %w", err)
	}
	return out, total, nil
}

// Create inserts a new stock.
func (r *StockRepository) Create(ctx context.Context, s domain.Stock) error {
	const q = `
		INSERT INTO stocks (symbol, name, sector_id, exchange, icon_url, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	if _, err := r.q.Exec(ctx, q,
		s.Symbol, s.Name, s.SectorID, s.Exchange, s.IconURL, s.IsActive,
	); err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("stock create: %w", domain.ErrStockSymbolTaken)
		}
		return fmt.Errorf("stock create: %w", err)
	}
	return nil
}

// Update applies a partial patch to the stock row.
func (r *StockRepository) Update(ctx context.Context, symbol string, p domain.StockPatch) error {
	var (
		sets []string
		args []any
	)
	if p.Name != nil {
		args = append(args, *p.Name)
		sets = append(sets, fmt.Sprintf("name = $%d", len(args)))
	}
	if p.SectorSet {
		args = append(args, p.SectorID)
		sets = append(sets, fmt.Sprintf("sector_id = $%d", len(args)))
	}
	if p.Exchange != nil {
		args = append(args, emptyToNil(*p.Exchange))
		sets = append(sets, fmt.Sprintf("exchange = $%d", len(args)))
	}
	if p.IconURL != nil {
		args = append(args, emptyToNil(*p.IconURL))
		sets = append(sets, fmt.Sprintf("icon_url = $%d", len(args)))
	}
	if p.IsActive != nil {
		args = append(args, *p.IsActive)
		sets = append(sets, fmt.Sprintf("is_active = $%d", len(args)))
	}
	if len(sets) == 0 {
		return nil // no-op
	}
	args = append(args, symbol)
	q := `UPDATE stocks SET ` + strings.Join(sets, ", ") +
		`, updated_at = now() WHERE symbol = $` + fmt.Sprint(len(args)) +
		` AND deleted_at IS NULL`
	tag, err := r.q.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("stock update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("stock update: %w", domain.ErrStockNotFound)
	}
	return nil
}

// SoftDelete marks the stock row deleted. It refuses when active watchlist
// rows still reference the symbol: the watchlists.symbol FK only blocks hard
// deletes, so without this guard watchlists would keep pointing at an
// invisible stock.
func (r *StockRepository) SoftDelete(ctx context.Context, symbol string) error {
	const q = `UPDATE stocks SET deleted_at = now()
		WHERE symbol = $1 AND deleted_at IS NULL
		AND NOT EXISTS (
			SELECT 1 FROM watchlists
			WHERE watchlists.symbol = stocks.symbol
			AND watchlists.deleted_at IS NULL
		)`
	tag, err := r.q.Exec(ctx, q, symbol)
	if err != nil {
		return fmt.Errorf("stock soft delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		if r.hasActiveWatchlists(ctx, symbol) {
			return fmt.Errorf("stock soft delete: %w", domain.ErrStockHasWatchlists)
		}
		return fmt.Errorf("stock soft delete: %w", domain.ErrStockNotFound)
	}
	return nil
}

// hasActiveWatchlists reports whether any live watchlist row references the
// symbol. Errors are swallowed (false) so a watchlist lookup failure degrades
// to ErrStockNotFound rather than a false 409.
func (r *StockRepository) hasActiveWatchlists(ctx context.Context, symbol string) bool {
	var exists bool
	if err := r.q.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM watchlists WHERE symbol = $1 AND deleted_at IS NULL)`,
		symbol,
	).Scan(&exists); err != nil {
		return false
	}
	return exists
}

// Upsert inserts or updates sync-owned fields for a symbol.
func (r *StockRepository) Upsert(ctx context.Context, s domain.Stock) (bool, error) {
	const q = `
		INSERT INTO stocks (symbol, name, icon_url, is_active, synced_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (symbol) DO UPDATE
		SET name = EXCLUDED.name,
		    icon_url = EXCLUDED.icon_url,
		    is_active = EXCLUDED.is_active,
		    synced_at = EXCLUDED.synced_at,
		    deleted_at = NULL,
		    updated_at = now()
		WHERE stocks.deleted_at IS NOT NULL
		   OR stocks.name IS DISTINCT FROM EXCLUDED.name
		   OR stocks.icon_url IS DISTINCT FROM EXCLUDED.icon_url
		   OR stocks.is_active IS DISTINCT FROM EXCLUDED.is_active
		RETURNING (xmax = 0) AS created
	`
	var created bool
	err := r.q.QueryRow(ctx, q, s.Symbol, s.Name, s.IconURL, s.IsActive).Scan(&created)
	if err != nil {
		// ON CONFLICT ... WHERE matched nothing (row unchanged): no row is
		// returned. Treat as a no-op, not an error.
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("stock upsert: %w", err)
	}
	return created, nil
}

// emptyToNil maps an empty string to nil so callers can clear a nullable
// column by sending "".
func emptyToNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullStringPtr converts a valid sql.NullString into a *string (nil when the
// column was NULL).
func nullStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	v := ns.String
	return &v
}
