CREATE TABLE stocks (
    symbol     TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    sector     TEXT,
    exchange   TEXT,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    synced_at  TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
-- Note: a separate idx_stocks_sector is intentionally NOT created. The
-- admin search endpoint combines sector with is_active + ILIKE, and the
-- partial idx_stocks_active below is a better fit for the dominant
-- read path (active user-facing list). If sector-only reports are added
-- later, add the index at that time with a real query plan to justify it.
CREATE INDEX idx_stocks_active ON stocks (is_active) WHERE deleted_at IS NULL;

-- Auto-refresh updated_at on every row UPDATE. The application also sets
-- it explicitly, but the trigger is the safety net.
CREATE TRIGGER trg_stocks_updated_at
    BEFORE UPDATE ON stocks
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Seed permissions (idempotent: re-running this migration is a no-op).
INSERT INTO permissions (resource, action, name) VALUES
    ('stocks', 'read',  'stocks:read'),
    ('stocks', 'write', 'stocks:write'),
    ('stocks', 'sync',  'stocks:sync')
ON CONFLICT (name) DO NOTHING;

-- Link user role to stocks:read only (idempotent).
INSERT INTO role_permissions (role_id, permission_id)
 SELECT r.id, p.id FROM roles r, permissions p
  WHERE r.name = 'user' AND p.name = 'stocks:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Link admin role to all three stock permissions (idempotent).
-- The admin role already gets all permissions from migration 000002
-- (its grant is "all"), but we re-link explicitly here to make the
-- dependency between this migration and 000002 obvious.
INSERT INTO role_permissions (role_id, permission_id)
 SELECT r.id, p.id FROM roles r, permissions p
  WHERE r.name = 'admin'
    AND p.name IN ('stocks:read', 'stocks:write', 'stocks:sync')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Backfill (idempotent): for every distinct symbol currently referenced
-- by a non-deleted watchlist row, make sure a corresponding row exists
-- in stocks. Inserted rows are inactive (is_active=false) so they don't
-- show up in the user-facing stock list — they're placeholders for the
-- relation, not real catalog entries. Admins can promote them later
-- (or let the next /stocks/sync replace them).
INSERT INTO stocks (symbol, name, is_active)
 SELECT DISTINCT w.symbol, 'Unknown', false
   FROM watchlists w
   WHERE w.deleted_at IS NULL
     AND NOT EXISTS (SELECT 1 FROM stocks s WHERE s.symbol = w.symbol)
ON CONFLICT (symbol) DO NOTHING;

-- Attach the FK to the existing watchlists.symbol column. The column
-- name is unchanged; the backfill above guarantees every watchlist
-- row already references a valid stocks row.
ALTER TABLE watchlists
    ADD CONSTRAINT watchlists_symbol_fkey
    FOREIGN KEY (symbol) REFERENCES stocks (symbol)
    ON DELETE RESTRICT;
