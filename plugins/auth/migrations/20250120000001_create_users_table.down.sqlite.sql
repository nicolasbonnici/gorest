-- Rollback users table creation
DROP INDEX IF EXISTS idx_user_email;
DROP TABLE IF EXISTS users;
