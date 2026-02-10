ALTER TABLE users
ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE;

UPDATE users
SET updated_at = created_at
WHERE updated_at IS NULL;

ALTER TABLE users
ALTER COLUMN updated_at
SET DEFAULT NOW();

ALTER TABLE users
ALTER COLUMN updated_at
SET NOT NULL;
