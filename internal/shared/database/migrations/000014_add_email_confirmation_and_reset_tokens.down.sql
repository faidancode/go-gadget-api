DROP TABLE IF EXISTS email_confirmation_tokens;
DROP TABLE IF EXISTS password_reset_tokens;

ALTER TABLE users
DROP COLUMN IF EXISTS email_confirmed;

