-- Drop the watchlist FK first. The watchlists.symbol column itself is
-- unchanged from 000003 — the FK is the only thing 000004 added there.
ALTER TABLE watchlists
    DROP CONSTRAINT watchlists_symbol_fkey;

-- Now reverse the stocks side.
DROP TRIGGER IF EXISTS trg_stocks_updated_at ON stocks;
DROP INDEX IF EXISTS idx_stocks_active;
DROP TABLE IF EXISTS stocks;

-- Remove the new permissions. role_permissions uses ON DELETE RESTRICT
-- on permissions, so clear the links first to avoid a FK violation.
DELETE FROM role_permissions
 WHERE permission_id IN (SELECT id FROM permissions WHERE name IN ('stocks:read', 'stocks:write', 'stocks:sync'));
DELETE FROM permissions WHERE name IN ('stocks:read', 'stocks:write', 'stocks:sync');

-- The backfilled placeholder stocks (is_active=false, name='Unknown')
-- are left in place — they're cheap, and removing them on rollback
-- would surprise anyone who already promoted them to real catalog
-- entries. An admin can clean them up explicitly.