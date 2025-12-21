-- Rollback users table creation
DROP INDEX idx_user_email ON users;
DROP TABLE IF EXISTS users;
