CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);
CREATE TABLE permissions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource   TEXT NOT NULL,
    action     TEXT NOT NULL,
    name       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    -- Each (resource, action) pair describes exactly one capability. The
    -- separate UNIQUE on name keeps the human-readable label unique too.
    CONSTRAINT permissions_resource_action_key UNIQUE (resource, action)
);
CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE RESTRICT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ,
    PRIMARY KEY (role_id, permission_id)
);
CREATE TABLE user_roles (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, role_id)
);
-- Note: idx_user_roles_user and idx_role_permissions_role are intentionally
-- NOT created; the composite primary keys already index the leading column
-- (user_id and role_id respectively) and a separate index would be dead
-- weight on writes.

-- Reuse the function from 000001 and attach triggers to every table that
-- has an updated_at column.
CREATE TRIGGER trg_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_permissions_updated_at
    BEFORE UPDATE ON permissions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_role_permissions_updated_at
    BEFORE UPDATE ON role_permissions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_user_roles_updated_at
    BEFORE UPDATE ON user_roles
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Seed permissions (idempotent: re-running this migration is a no-op)
INSERT INTO permissions (resource, action, name) VALUES
 ('auth','login','auth:login'),
 ('profile','read','profile:read'),
 ('profile','write','profile:write'),
 ('watchlist','read','watchlist:read'),
 ('watchlist','write','watchlist:write'),
 ('admin','roles:read','admin:roles:read'),
 ('admin','roles:write','admin:roles:write'),
 ('admin','users:read','admin:users:read'),
 ('admin','users:manage','admin:users:manage'),
 ('admin','rbac:assign','admin:rbac:assign')
ON CONFLICT (name) DO NOTHING;

-- Seed roles (idempotent)
INSERT INTO roles (name, description) VALUES
 ('user','Default end user'),
 ('admin','Full administrator')
ON CONFLICT (name) DO NOTHING;

-- Link user role to its permissions (idempotent)
INSERT INTO role_permissions (role_id, permission_id)
 SELECT r.id, p.id FROM roles r, permissions p
  WHERE r.name='user' AND p.name IN ('auth:login','profile:read','profile:write','watchlist:read','watchlist:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Link admin role to all permissions (idempotent)
INSERT INTO role_permissions (role_id, permission_id)
 SELECT r.id, p.id FROM roles r, permissions p WHERE r.name='admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;
