-- sectors is the master table for stock sectors ("Financials", "Technology",
-- ...). It is created BEFORE stocks (000004) so stocks.sector_id can carry a
-- real foreign key.
--
-- Sectors are managed manually for now (admin/ops insert rows directly — no
-- seed and no auto-upsert from upstream yet). stocks.sector_id is nullable,
-- so a stock may exist without a sector until one is assigned by hand.
CREATE TABLE sectors (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
-- Note: idx_sectors_deleted_at is intentionally NOT created — with manual
-- management the table stays tiny and every read filters by name/id anyway.

-- Auto-refresh updated_at on every row UPDATE. The application also sets
-- it explicitly, but the trigger is the safety net.
CREATE TRIGGER trg_sectors_updated_at
    BEFORE UPDATE ON sectors
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
