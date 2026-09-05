DROP TRIGGER IF EXISTS trg_watchlists_updated_at ON watchlists;
DROP TABLE IF EXISTS watchlists;
-- Note: set_updated_at() is intentionally NOT dropped here. The function is
-- owned by migration 000001 and is still used by the users/roles/etc.
-- triggers until those tables are rolled back by their own migrations.
