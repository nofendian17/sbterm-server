package domain

import "time"

// Sector is one row in the manually-managed sectors master table. Stocks
// reference it by SectorID; responses usually also carry the joined Name.
type Sector struct {
	ID   string
	Name string
}

// Stock is one row in the stocks master catalog. Sector is populated by
// reads that LEFT JOIN sectors; writes only set SectorID.
type Stock struct {
	Symbol    string
	Name      string
	SectorID  *string
	Sector    *Sector
	Exchange  *string
	IconURL   *string
	IsActive  bool
	SyncedAt  *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StockFilter narrows StockRepository.List. Empty fields mean "no filter".
// Sector matches the sector *name* (exact) via a join to sectors. IsActive
// is a pointer so callers can distinguish "unset" from "explicitly false".
type StockFilter struct {
	Query    string // ILIKE on symbol or name
	Sector   string // exact sector name
	IsActive *bool
	Page     int
	Limit    int
}

// NormalizePagination applies the module-wide defaults (page=1, limit=20,
// cap 100), matching the existing admin user list. It is the single shared
// helper so handlers (response meta) and repositories (SQL LIMIT/OFFSET)
// never drift apart.
func NormalizePagination(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

// StockPatch is the set of fields a PATCH may change. A nil pointer means
// "leave alone". To clear a nullable column the caller passes a pointer to
// the empty string, which the repository stores as NULL.
type StockPatch struct {
	Name      *string
	SectorID  *string
	SectorSet bool // when true, set sector_id to SectorID (nil clears it)
	Exchange  *string
	IconURL   *string
	IsActive  *bool
}

// StockCreateInput is what the admin create flow accepts. Sector is the
// sector *name* (optional); an empty value means "no sector". The usecase
// resolves the name to a sector id before persisting.
type StockCreateInput struct {
	Symbol   string
	Name     string
	Sector   *string
	Exchange *string
	IconURL  *string
	IsActive *bool
}

// StockUpdateInput is what the admin update flow accepts. A nil Sector means
// "unchanged"; a pointer to "" clears the sector. Name/Exchange/IconURL nil
// mean unchanged and "" clears the nullable ones (name must stay non-empty).
type StockUpdateInput struct {
	Name     *string
	Sector   *string
	Exchange *string
	IconURL  *string
	IsActive *bool
}

// StockSyncResult is the outcome of a catalog sync from the upstream
// apps/api endpoint. Errors lists per-symbol failures; the overall error is
// only returned when the upstream call itself failed.
type StockSyncResult struct {
	Fetched int
	Created int
	Updated int
	Skipped int
	Failed  int
	Errors  []string
}
