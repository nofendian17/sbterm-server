-- Single-migration rollback: drop child tables first (removing FKs to
-- company_profiles), then the profile header.
DROP TABLE IF EXISTS company_addresses;
DROP TABLE IF EXISTS company_subsidiaries;
DROP TABLE IF EXISTS company_shareholder_numbers;
DROP TABLE IF EXISTS company_holdings;
DROP TABLE IF EXISTS company_executives;
DROP TABLE IF EXISTS company_profiles;
