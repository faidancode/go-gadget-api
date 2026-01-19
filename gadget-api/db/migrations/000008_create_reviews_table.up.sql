CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT NOT NULL,
    is_verified_purchase BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT uniq_user_product_review UNIQUE(user_id, product_id)
);

CREATE INDEX idx_reviews_product ON reviews(product_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_reviews_user ON reviews(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_reviews_rating ON reviews(rating) WHERE deleted_at IS NULL;