CREATE TABLE boards (
                        id SERIAL PRIMARY KEY,
                        title TEXT,
                        description TEXT,
                        owner_id INTEGER REFERENCES users(id),
                        created_at TIMESTAMP,
                        updated_at TIMESTAMP,
                        deleted_at TIMESTAMP
);