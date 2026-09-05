-- Company profile cluster: every stock MAY have one company profile
-- (1:1 optional; stocks is the master). Columns and child tables below are
-- aligned to the apps/api payloads in docs/api.md:
--   - company_profiles        <- GET /api/v1/company/{symbol}/profile (scalars)
--   - company_executives      <- profile.key_executive
--   - company_holdings        <- profile.shareholder / .shareholder_one_percent /
--                                 .shareholder_director_commissioner / .beneficiary
--   - company_shareholder_numbers <- profile.shareholder_numbers (time series)
--   - company_subsidiaries    <- GET /api/v1/company/{symbol}/subsidiaries (rich)
--   - company_addresses       <- profile.address
--
-- Access to profile data rides on the existing stocks:read / stocks:write
-- permissions — no new RBAC seeds are needed.

-- ---------------------------------------------------------------------------
-- 1:1 profile header. PK = symbol enforces "at most one profile per stock".
-- The upstream profile groups scalar history facts (board, listing date/price,
-- IPO amount, listed shares, free float, registrar); they are kept as TEXT to
-- mirror the formatted strings the API returns, so the sync layer does not
-- need to parse every value.
-- ---------------------------------------------------------------------------
CREATE TABLE company_profiles (
    symbol       TEXT PRIMARY KEY REFERENCES stocks (symbol) ON DELETE RESTRICT,
    background   TEXT,              -- profile.background
    board        TEXT,              -- profile.history.board, e.g. "Papan Utama"
    listing_date TEXT,              -- profile.history.date, e.g. "31 May 2000"
    listing_price TEXT,             -- profile.history.price, e.g. "1,400"
    ipo_amount   TEXT,              -- profile.history.amount, e.g. "927 B"
    listed_shares TEXT,             -- profile.history.shares, e.g. "662,400,000"
    free_float   TEXT,              -- profile.history.free_float, e.g. "42.46%"
    registrar    TEXT,              -- profile.history.registrar
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);
-- ON DELETE RESTRICT is consistent with the whole schema: stocks are never
-- hard-deleted while referenced; removal uses soft delete.

CREATE TRIGGER trg_company_profiles_updated_at
    BEFORE UPDATE ON company_profiles
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- Child tables (fully normalized). Every child row: id UUID PK + symbol FK
-- to company_profiles, soft delete, updated_at trigger.
-- ---------------------------------------------------------------------------

-- Board of commissioners & directors from profile.key_executive. kind follows
-- the upstream group: commissioner / director / independent_commissioner.
-- role = the "key" label, position = upstream display order,
-- external_id = upstream id.
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

-- Shareholders & related entities, distinguished by holder_group. percentage
-- is stored twice (parsed NUMERIC + raw upstream text), same for
-- value/amount; small badges stay a per-row label array.
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

-- Shareholder-count time series per report (profile.shareholder_numbers):
-- one row per shareholder_date. Text columns follow the upstream format.
CREATE TABLE company_shareholder_numbers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol          TEXT NOT NULL REFERENCES company_profiles (symbol) ON DELETE RESTRICT,
    shareholder_date TEXT NOT NULL,   -- e.g. "30 Jun 2026"
    total_share     TEXT,             -- e.g. "86,926"
    change          BIGINT,           -- holder-count delta
    change_formatted TEXT,            -- e.g. "(+48,123)"
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE (symbol, shareholder_date)
);

-- Subsidiaries from the dedicated GET /api/v1/company/{symbol}/subsidiaries
-- endpoint (richer than profile.subsidiary). total_assets & percentage are
-- parsed numerics + raw upstream. currency/unit/last_updated_period are
-- response-level properties (not per-row) and are not stored here.
CREATE TABLE company_subsidiaries (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol           TEXT NOT NULL REFERENCES company_profiles (symbol) ON DELETE RESTRICT,
    name             TEXT NOT NULL,   -- subsidiaries[].company_name
    business_type    TEXT,
    location         TEXT,
    commercial_year  TEXT,
    total_assets     NUMERIC(20, 2),
    total_assets_raw TEXT,
    percentage       NUMERIC(12, 4),
    percentage_raw   TEXT,
    operational_status TEXT,
    period           TEXT,
    position         INT NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

-- Address/contact from profile.address; small flat emails -> TEXT[] array.
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

-- Dominant child-table read path = "all rows of one profile" (WHERE symbol = $1);
-- FKs don't create indexes automatically, so the btree on symbol is explicit.
CREATE INDEX idx_company_executives_symbol ON company_executives (symbol) WHERE deleted_at IS NULL;
CREATE INDEX idx_company_holdings_symbol ON company_holdings (symbol) WHERE deleted_at IS NULL;
CREATE INDEX idx_company_shareholder_numbers_symbol ON company_shareholder_numbers (symbol) WHERE deleted_at IS NULL;
CREATE INDEX idx_company_subsidiaries_symbol ON company_subsidiaries (symbol) WHERE deleted_at IS NULL;
CREATE INDEX idx_company_addresses_symbol ON company_addresses (symbol) WHERE deleted_at IS NULL;
