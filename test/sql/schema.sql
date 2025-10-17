DROP TABLE IF EXISTS todo CASCADE;
DROP TABLE IF EXISTS users CASCADE;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    firstname TEXT NOT NULL,
    lastname TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password TEXT,
    updated_at TIMESTAMP(0) WITHOUT TIME ZONE,
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_email ON users (email);

CREATE TABLE todo (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    updated_at TIMESTAMP(0) WITHOUT TIME ZONE,
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_todo_title ON todo (title);
CREATE INDEX idx_todo_fk_user ON todo (user_id);


-- Fixtures
WITH gen AS (SELECT gen_random_uuid() AS uuid)
INSERT INTO users (id, firstname, lastname, email, password)
SELECT gen.uuid, 'Admin', 'User', 'admin@test.com', encode(digest('salt' || 'password' || gen.uuid::text, 'sha256'), 'hex')
FROM gen;