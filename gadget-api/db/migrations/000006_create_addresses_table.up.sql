CREATE TABLE addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    label VARCHAR(60) NOT NULL,
    recipient_name VARCHAR(120) NOT NULL,
    recipient_phone VARCHAR(30) NOT NULL,
    street VARCHAR(255) NOT NULL,
    subdistrict VARCHAR(120),
    district VARCHAR(120),
    city VARCHAR(120),
    province VARCHAR(120),
    postal_code VARCHAR(20),
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_addresses_user_id ON addresses (user_id);
CREATE INDEX idx_addresses_user_primary ON addresses (user_id, is_primary);
