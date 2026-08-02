DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS todo CASCADE;
DROP TABLE IF EXISTS users CASCADE;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    firstname TEXT NOT NULL,
    lastname TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password TEXT,
    updated_at TIMESTAMP(0) WITH TIME ZONE,
    created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_email ON users (email);

CREATE TABLE todo (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    updated_at TIMESTAMP(0) WITH TIME ZONE,
    created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_todo_title ON todo (title);
CREATE INDEX idx_todo_fk_user ON todo (user_id);


-- Fixtures
-- Test user: admin@test.com / password (bcrypt hash)
INSERT INTO users (firstname, lastname, email, password)
VALUES ('Admin', 'User', 'admin@test.com', '$2a$10$xZybcXcww7epzFX6d6yr1uWKJvnqs7cEySXCKDYlBN1frJeUswGla');