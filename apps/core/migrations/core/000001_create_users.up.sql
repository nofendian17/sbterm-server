CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    expires_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);
-- Note: an idx_users_email is intentionally NOT created; the UNIQUE
-- constraint on email already builds a btree, and a separate index would
-- duplicate the work.
CREATE INDEX idx_users_deleted_at ON users (deleted_at);
CREATE INDEX idx_users_expires_at ON users (expires_at);

-- Auto-refresh updated_at on every row UPDATE. The application also sets it
-- explicitly, but the trigger is the safety net.
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
