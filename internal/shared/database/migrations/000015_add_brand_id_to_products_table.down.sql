DROP INDEX IF EXISTS idx_products_brand_id;

ALTER TABLE products
DROP COLUMN IF EXISTS brand_id;
