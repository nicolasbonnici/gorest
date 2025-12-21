-- Create users table for authentication
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
