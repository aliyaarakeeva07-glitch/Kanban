CREATE TABLE users (
                       id SERIAL PRIMARY KEY,
                       name TEXT,
                       email TEXT UNIQUE,
                       password_hash TEXT,
                       created_at TIMESTAMP,
                       updated_at TIMESTAMP
);