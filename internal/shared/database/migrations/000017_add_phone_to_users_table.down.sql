-- Down: Remove phone field from users table
ALTER TABLE users 
DROP COLUMN phone;