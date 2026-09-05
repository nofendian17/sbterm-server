-- watchlists references two masters: users (owner) and stocks (catalog).
-- stocks is created in 000004, so the FK below is inline — no backfill or
-- ALTER is needed. A symbol must already exist in stocks (active or not)
-- before any watchlist row can reference it; soft-deleted stock rows still
-- satisfy the FK, only a hard delete is blocked (ON DELETE RESTRICT).
CREATE TABLE watchlists (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    symbol     TEXT NOT NULL REFERENCES stocks(symbol) ON DELETE RESTRICT,
    label      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (user_id, symbol)
);
-- Note: idx_watchlists_user_id is intentionally NOT created; the UNIQUE
-- constraint (user_id, symbol) already builds a btree with user_id as the
-- leading column, which is exactly what a separate index would cover.

-- Auto-refresh updated_at on every row UPDATE. The application also sets
-- it explicitly, but the trigger is the safety net.
CREATE TRIGGER trg_watchlists_updated_at
    BEFORE UPDATE ON watchlists
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
