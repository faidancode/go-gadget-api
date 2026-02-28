ALTER TABLE products
ADD COLUMN brand_id UUID REFERENCES brands(id);

CREATE INDEX idx_products_brand_id ON products(brand_id);
