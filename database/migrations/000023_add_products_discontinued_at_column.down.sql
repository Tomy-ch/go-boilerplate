DROP INDEX IF EXISTS products_discontinued_at_idx;

ALTER TABLE products
DROP COLUMN IF EXISTS discontinued_at;
