ALTER TABLE orders
ADD COLUMN address_id UUID;

ALTER TABLE orders
ADD CONSTRAINT fk_orders_address
FOREIGN KEY (address_id)
REFERENCES addresses(id)
ON UPDATE CASCADE
ON DELETE SET NULL;

CREATE INDEX idx_orders_address_id
ON orders(address_id);
