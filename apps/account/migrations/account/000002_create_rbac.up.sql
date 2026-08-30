CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE permissions (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource TEXT NOT NULL,
    action   TEXT NOT NULL,
    name     TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX idx_user_roles_user ON user_roles (user_id);
CREATE INDEX idx_role_permissions_role ON role_permissions (role_id);

-- Seed permissions
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
 ('admin','rbac:assign','admin:rbac:assign');

-- Seed roles
INSERT INTO roles (name, description) VALUES ('user','Default end user'), ('admin','Full administrator');
INSERT INTO role_permissions (role_id, permission_id)
 SELECT r.id, p.id FROM roles r, permissions p
  WHERE r.name='user' AND p.name IN ('auth:login','profile:read','profile:write','watchlist:read','watchlist:write');
INSERT INTO role_permissions (role_id, permission_id)
 SELECT r.id, p.id FROM roles r, permissions p WHERE r.name='admin';
