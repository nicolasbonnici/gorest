DROP TABLE IF EXISTS todo;
DROP TABLE IF EXISTS users;

CREATE TABLE users (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    firstname TEXT NOT NULL,
    lastname TEXT NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password TEXT,
    updated_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_email ON users (email);

CREATE TABLE todo (
    id CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id CHAR(36),
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    updated_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_todo_title ON todo (title(255));
CREATE INDEX idx_todo_fk_user ON todo (user_id);

-- Fixtures
-- Test user: admin@test.com / password (bcrypt hash)
INSERT INTO users (id, firstname, lastname, email, password)
VALUES (UUID(), 'Admin', 'User', 'admin@test.com', '$2a$10$xZybcXcww7epzFX6d6yr1uWKJvnqs7cEySXCKDYlBN1frJeUswGla');
