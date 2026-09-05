-- =============================================================================
-- apps/core — Rancangan Database (blueprint konsolidasi)
-- =============================================================================
-- Cakupan      : database Postgres milik apps/core (sbterm-core, port :8082)
-- Sumber       : migrations/core/000001..000006 (golang-migrate, run at startup)
--                Kolom katalog/profile diselaraskan dengan payload docs/api.md
--                (endpoint apps/api yang menjadi sumber sync ke depannya).
-- Tujuan       : dokumen desain tunggal yang memperlihatkan seluruh tabel dan
--                relasi antarentitas, tanpa harus membaca 6 file migration.
-- Pemeliharaan : ditulis tangan, bukan hasil generate; setiap perubahan
--                migrations/core/00*.up.sql wajib dipantulkan ke sini.
--
-- Relasi inti (1..*, 1..1):
--
--   users ──< user_roles >── roles ──< role_permissions >── permissions
--   users ──< watchlists >── stocks ──< sectors
--   stocks ──(1:1 opsional)── company_profiles ──< company_executives
--                                             ──< company_holdings
--                                             ──< company_shareholder_numbers
--                                             ──< company_subsidiaries
--                                             ──< company_addresses
--
--   - 1 user memiliki banyak role (user_roles), 1 role dimiliki banyak user.
--   - 1 role memiliki banyak permission (role_permissions), 1 permission
--     dipakai banyak role (relasi many-to-many dengan tabel penghubung).
--   - 1 user memiliki banyak baris watchlist (watchlists.user_id).
--   - 1 baris watchlist merujuk tepat 1 simbol di katalog stocks
--     (watchlists.symbol), sedangkan 1 simbol stocks bisa ada di banyak
--     watchlist milik banyak user.
--   - Banyak stocks berada dalam 1 sektor (stocks.sector_id). N-ke-1 OPSIONAL:
--     sektor dikelola manual dulu (sumber nama/isi sektor menyusul).
--   - 1 stock memiliki Paling Banyak 1 company profile (company_profiles
--     ber-PK symbol → 1:1). Bagian kompleksnya dinormalisasi penuh dan
--     kolomnya meniru payload apps/api (docs/api.md).
--
-- Sumber sync (docs/api.md) → tabel:
--   GET /api/v1/stocks                        → stocks (symbol, name, icon_url,
--                                                is_active dari company_status)
--   GET /api/v1/sectors                       → sectors (kode index + membership)
--   GET /api/v1/company/{symbol}/profile      → company_profiles, executives,
--                                                holdings, shareholder_numbers, addresses
--   GET /api/v1/company/{symbol}/subsidiaries → company_subsidiaries (detail kaya)
--
-- Konvensi yang berlaku di seluruh schema:
--   - Soft delete: kolom deleted_at di tabel utama; semua query baca
--     memfilter deleted_at IS NULL.
--   - updated_at di-refresh otomatis oleh trigger set_updated_at().
--   - Tidak ada index redundan: UNIQUE/PK constraint sudah membangun btree
--     yang mencakup kolom leading yang sama.
--   - Seed (roles/permissions/grants) idempotent via ON CONFLICT DO NOTHING.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. users
-- -----------------------------------------------------------------------------
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    expires_at    TIMESTAMPTZ,                -- per-user expiry (diperiksa server-side)
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ                 -- soft delete / suspensi akun
);

-- Index terarah untuk jalur baca dominan: akun kedaluwarsa & terhapus.
CREATE INDEX idx_users_deleted_at ON users (deleted_at);
CREATE INDEX idx_users_expires_at ON users (expires_at);
-- Catatan: idx_users_email sengaja TIDAK dibuat — UNIQUE(email) sudah membangun
-- btree yang sama; index terpisah hanya duplikasi beban tulis.

-- Trigger bersama untuk auto-refresh updated_at (dipakai semua tabel).
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

-- -----------------------------------------------------------------------------
-- 2. RBAC: roles, permissions, dan dua tabel penghubung many-to-many
-- -----------------------------------------------------------------------------
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
    name       TEXT NOT NULL UNIQUE,          -- label "<resource>:<action>", mis. admin:rbac:assign
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT permissions_resource_action_key UNIQUE (resource, action)
);

-- Penghubung role -> permissions (many-to-many).
CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles (id) ON DELETE RESTRICT,
    permission_id UUID NOT NULL REFERENCES permissions (id) ON DELETE RESTRICT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ,
    PRIMARY KEY (role_id, permission_id)
);

-- Penghubung users -> roles (many-to-many).
CREATE TABLE user_roles (
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    role_id    UUID NOT NULL REFERENCES roles (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, role_id)
);

-- ON DELETE RESTRICT (bukan CASCADE): role/permission tidak bisa dihapus
-- selama masih dirujuk baris penghubung — hapus link-nya dulu secara eksplisit.
-- Catatan: idx_user_roles_user & idx_role_permissions_role sengaja tidak dibuat;
-- PK komposit sudah meng-index kolom leading (user_id / role_id).

CREATE TRIGGER trg_roles_updated_at
    BEFORE UPDATE ON roles FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_permissions_updated_at
    BEFORE UPDATE ON permissions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_role_permissions_updated_at
    BEFORE UPDATE ON role_permissions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_user_roles_updated_at
    BEFORE UPDATE ON user_roles FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- -----------------------------------------------------------------------------
-- 3. sectors (master sektor saham; tabel induk untuk stocks.sector_id)
-- -----------------------------------------------------------------------------
CREATE TABLE sectors (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL UNIQUE,          -- kode index sektor: IDXBASIC, IDXCYCLIC, ...
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- Sumber kandidat: GET /api/v1/sectors (index + icon + konstituen). Dikelola
-- manual untuk saat ini; membership nanti mengisi stocks.sector_id.
CREATE TRIGGER trg_sectors_updated_at
    BEFORE UPDATE ON sectors FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- -----------------------------------------------------------------------------
-- 4. stocks (katalog saham; tabel induk watchlists & company_profiles)
-- -----------------------------------------------------------------------------
CREATE TABLE stocks (
    symbol     TEXT PRIMARY KEY,              -- ticker Stockbit: BBCA, TLKM, ...
    name       TEXT NOT NULL,
    sector_id  UUID REFERENCES sectors (id) ON DELETE RESTRICT,  -- nullable, kelola manual
    exchange   TEXT,
    icon_url   TEXT,                          -- logo; dari payload GET /api/v1/stocks
    is_active  BOOLEAN NOT NULL DEFAULT true, -- sync: company_status = STATUS_ACTIVE
    synced_at  TIMESTAMPTZ,                   -- waktu sync upstream terakhir
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- Jalur baca dominan (list aktif untuk user) → partial index terarah.
CREATE INDEX idx_stocks_active ON stocks (is_active) WHERE deleted_at IS NULL;
-- Catatan: index per-sektor di stocks sengaja tidak dibuat — filter sektor
-- di-join ke sectors yang sudah ter-index oleh UNIQUE(name).

CREATE TRIGGER trg_stocks_updated_at
    BEFORE UPDATE ON stocks FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- -----------------------------------------------------------------------------
-- 5. watchlists (milik user, merujuk simbol di stocks)
-- -----------------------------------------------------------------------------
CREATE TABLE watchlists (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    symbol     TEXT NOT NULL REFERENCES stocks (symbol) ON DELETE RESTRICT,
    label      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (user_id, symbol)                  -- 1 user tidak bisa dobel 1 simbol
);

-- Catatan: idx_watchlists_user_id sengaja tidak dibuat — UNIQUE(user_id, symbol)
-- sudah meng-index user_id sebagai kolom leading.
-- ON DELETE RESTRICT: simbol stocks tidak bisa dihapus selama masih dipakai
-- watchlist; hapus/soft-delete referensinya dulu (soft-delete stocks tetap
-- membuat baris "tak terlihat", jadi RESTRICT hanya menahan hard delete).

CREATE TRIGGER trg_watchlists_updated_at
    BEFORE UPDATE ON watchlists FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- -----------------------------------------------------------------------------
-- 6. company_profiles (1:1 opsional dengan stocks) + tabel anak ternormalisasi
--    Kolom meniru payload docs/api.md.
-- -----------------------------------------------------------------------------
-- Header: skalar + fakta history dari profile. Nilai numerik upstream berupa
-- string berformat, sehingga disimpan TEXT agar layer sync tidak wajib parse.
CREATE TABLE company_profiles (
    symbol        TEXT PRIMARY KEY REFERENCES stocks (symbol) ON DELETE RESTRICT,
    background    TEXT,              -- profile.background
    board         TEXT,              -- history.board ("Papan Utama")
    listing_date  TEXT,              -- history.date ("31 May 2000")
    listing_price TEXT,              -- history.price ("1,400")
    ipo_amount    TEXT,              -- history.amount ("927 B")
    listed_shares TEXT,              -- history.shares ("662,400,000")
    free_float    TEXT,              -- history.free_float ("42.46%")
    registrar     TEXT,              -- history.registrar
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

-- PK = symbol → "paling banyak satu profil per stock". Opsional: baris hanya
-- ada jika profil dibuat.
CREATE TRIGGER trg_company_profiles_updated_at
    BEFORE UPDATE ON company_profiles FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Dewan komisaris & direksi — profile.key_executive.
-- kind: commissioner / director / independent_commissioner; role = label "key".
CREATE TABLE company_executives (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol      TEXT NOT NULL REFERENCES company_profiles (symbol) ON DELETE RESTRICT,
    kind        TEXT NOT NULL CHECK (kind IN ('commissioner', 'director', 'independent_commissioner')),
    name        TEXT NOT NULL,
    role        TEXT,
    external_id TEXT,
    position    INT  NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

-- Pemegang & entitas terkait — satu tabel, dibedakan holder_group:
--   shareholder                profile.shareholder
--   one_percent                profile.shareholder_one_percent
--   director_commissioner      profile.shareholder_director_commissioner
--   beneficiary                profile.beneficiary
-- percentage/amount: NUMERIC hasil parse + *_raw teks mentah; badges array.
CREATE TABLE company_holdings (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol         TEXT NOT NULL REFERENCES company_profiles (symbol) ON DELETE RESTRICT,
    holder_group   TEXT NOT NULL CHECK (holder_group IN
                     ('shareholder', 'one_percent', 'director_commissioner', 'beneficiary')),
    name           TEXT NOT NULL,
    percentage     NUMERIC(12, 4),
    percentage_raw TEXT,
    amount_raw     TEXT,
    badges         TEXT[] NOT NULL DEFAULT '{}',
    position       INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ
);

-- Deret waktu jumlah pemegang per laporan — profile.shareholder_numbers.
CREATE TABLE company_shareholder_numbers (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol           TEXT NOT NULL REFERENCES company_profiles (symbol) ON DELETE RESTRICT,
    shareholder_date TEXT NOT NULL,     -- "30 Jun 2026"
    total_share      TEXT,              -- "86,926"
    change           BIGINT,            -- delta jumlah pemegang
    change_formatted TEXT,              -- "(+48,123)"
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ,
    UNIQUE (symbol, shareholder_date)
);

-- Anak perusahaan — GET /api/v1/company/{symbol}/subsidiaries (lebih kaya dari
-- profile.subsidiary). currency/unit/last_updated_period (properti respons)
-- tidak disimpan per-baris.
CREATE TABLE company_subsidiaries (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol             TEXT NOT NULL REFERENCES company_profiles (symbol) ON DELETE RESTRICT,
    name               TEXT NOT NULL,   -- company_name
    business_type      TEXT,
    location           TEXT,
    commercial_year    TEXT,
    total_assets       NUMERIC(20, 2),
    total_assets_raw   TEXT,
    percentage         NUMERIC(12, 4),
    percentage_raw     TEXT,
    operational_status TEXT,
    period             TEXT,
    position           INT NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ
);

-- Alamat/kontak — profile.address; emails kecil & flat → array TEXT[].
CREATE TABLE company_addresses (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol     TEXT NOT NULL REFERENCES company_profiles (symbol) ON DELETE RESTRICT,
    office     TEXT,
    phone      TEXT,
    fax        TEXT,
    website    TEXT,
    npwp       TEXT,
    emails     TEXT[] NOT NULL DEFAULT '{}',
    position   INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TRIGGER trg_company_executives_updated_at
    BEFORE UPDATE ON company_executives FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_company_holdings_updated_at
    BEFORE UPDATE ON company_holdings FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_company_shareholder_numbers_updated_at
    BEFORE UPDATE ON company_shareholder_numbers FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_company_subsidiaries_updated_at
    BEFORE UPDATE ON company_subsidiaries FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_company_addresses_updated_at
    BEFORE UPDATE ON company_addresses FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Jalur baca dominan tabel anak = "semua baris satu profil" (WHERE symbol = $1);
-- FK tidak otomatis membuat index, jadi btree di symbol dibuat eksplisit.
CREATE INDEX idx_company_executives_symbol ON company_executives (symbol) WHERE deleted_at IS NULL;
CREATE INDEX idx_company_holdings_symbol ON company_holdings (symbol) WHERE deleted_at IS NULL;
CREATE INDEX idx_company_shareholder_numbers_symbol ON company_shareholder_numbers (symbol) WHERE deleted_at IS NULL;
CREATE INDEX idx_company_subsidiaries_symbol ON company_subsidiaries (symbol) WHERE deleted_at IS NULL;
CREATE INDEX idx_company_addresses_symbol ON company_addresses (symbol) WHERE deleted_at IS NULL;

-- -----------------------------------------------------------------------------
-- 7. Seed RBAC (idempotent)
-- -----------------------------------------------------------------------------
INSERT INTO permissions (resource, action, name) VALUES
 ('auth','login','auth:login'),
 ('profile','read','profile:read'),
 ('profile','write','profile:write'),
 ('watchlist','read','watchlist:read'),
 ('watchlist','write','watchlist:write'),
 ('stocks','read','stocks:read'),
 ('stocks','write','stocks:write'),
 ('stocks','sync','stocks:sync'),
 ('admin','roles:read','admin:roles:read'),
 ('admin','roles:write','admin:roles:write'),
 ('admin','users:read','admin:users:read'),
 ('admin','users:manage','admin:users:manage'),
 ('admin','rbac:assign','admin:rbac:assign')
ON CONFLICT (name) DO NOTHING;

INSERT INTO roles (name, description) VALUES
 ('user','Default end user'),
 ('admin','Full administrator')
ON CONFLICT (name) DO NOTHING;

-- Role "user": permission self-service (auth, profile, watchlist, baca stocks).
INSERT INTO role_permissions (role_id, permission_id)
 SELECT r.id, p.id FROM roles r, permissions p
  WHERE r.name = 'user'
    AND p.name IN ('auth:login','profile:read','profile:write',
                   'watchlist:read','watchlist:write','stocks:read')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Role "admin": semua permission.
INSERT INTO role_permissions (role_id, permission_id)
 SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;
