-- No stocks FK is dropped here: stocks (000004) is rolled back first, and
-- dropping that table removes the FK referencing sectors. golang-migrate
-- always rolls back in descending version order.
DROP TRIGGER IF EXISTS trg_sectors_updated_at ON sectors;
DROP TABLE IF EXISTS sectors;
