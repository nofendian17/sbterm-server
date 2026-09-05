-- No watchlist FK is dropped here: watchlists (000005) is rolled back first,
-- and dropping its table removes the FK that references stocks. Similarly,
-- the sector FK is removed with this table; sectors itself is dropped by
-- 000003 afterwards.
DROP TRIGGER IF EXISTS trg_stocks_updated_at ON stocks;
DROP INDEX IF EXISTS idx_stocks_active;
DROP TABLE IF EXISTS stocks;

-- Remove the stock permissions. role_permissions uses ON DELETE RESTRICT
-- on permissions, so clear the links first to avoid a FK violation.
DELETE FROM role_permissions
 WHERE permission_id IN (SELECT id FROM permissions WHERE name IN ('stocks:read', 'stocks:write', 'stocks:sync'));
DELETE FROM permissions WHERE name IN ('stocks:read', 'stocks:write', 'stocks:sync');
