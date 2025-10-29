DROP TABLE IF EXISTS todo;
DROP TABLE IF EXISTS users;

CREATE TABLE users (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    firstname TEXT NOT NULL,
    lastname TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password TEXT,
    updated_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_email ON users (email);

CREATE TABLE todo (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    user_id TEXT,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    updated_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_todo_title ON todo (title);
CREATE INDEX idx_todo_fk_user ON todo (user_id);

-- Fixtures
-- Test user: admin@test.com / password (bcrypt hash)
INSERT INTO users (id, firstname, lastname, email, password)
VALUES ('admin-test-id', 'Admin', 'User', 'admin@test.com', '$2a$10$xZybcXcww7epzFX6d6yr1uWKJvnqs7cEySXCKDYlBN1frJeUswGla');
