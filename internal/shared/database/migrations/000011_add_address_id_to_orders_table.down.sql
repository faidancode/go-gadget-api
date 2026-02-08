-- 1. Drop index
DROP INDEX IF EXISTS idx_orders_address_id;

-- 2. Drop foreign key constraint
ALTER TABLE orders
DROP CONSTRAINT IF EXISTS fk_orders_address;

-- 3. Drop column
ALTER TABLE orders
DROP COLUMN IF EXISTS address_id;
