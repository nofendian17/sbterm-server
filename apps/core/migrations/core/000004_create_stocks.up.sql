-- stocks is the master catalog table. It is created BEFORE watchlists
-- (000005) so watchlists.symbol can carry a real inline foreign key with no
-- placeholder backfill — a symbol must exist here before any watchlist row
-- can reference it.
CREATE TABLE stocks (
    symbol     TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    -- Sector lives in its own master table (000003). sector_id is nullable:
    -- sectors are managed manually for now, so a stock may exist without one.
    sector_id  UUID REFERENCES sectors(id) ON DELETE RESTRICT,
    exchange   TEXT,
    -- Stock logo/icon, filled from the GET /api/v1/stocks payload (apps/api).
    -- company_status="STATUS_ACTIVE" in the same payload maps to is_active.
    icon_url   TEXT,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    synced_at  TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
-- Note: the partial index below covers the dominant read path (active
-- user-facing list, filtered by is_active + deleted_at). Sector-based
-- filtering joins sectors by name/id and is served by sectors' UNIQUE(name)
-- btree; no per-sector index on stocks is justified yet.
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

-- Link admin role to all three stock permissions (idempotent). The admin
-- role's "all permissions" grant from 000002 is a snapshot of the
-- permissions that existed then, so new permissions must be linked here
-- explicitly.
INSERT INTO role_permissions (role_id, permission_id)
 SELECT r.id, p.id FROM roles r, permissions p
  WHERE r.name = 'admin'
    AND p.name IN ('stocks:read', 'stocks:write', 'stocks:sync')
ON CONFLICT (role_id, permission_id) DO NOTHING;
