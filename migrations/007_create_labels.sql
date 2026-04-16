CREATE TABLE labels (
                        id SERIAL PRIMARY KEY,
                        board_id INTEGER REFERENCES boards(id),
                        name TEXT,
                        color TEXT,
                        created_at TIMESTAMP,
                        updated_at TIMESTAMP,
                        deleted_at TIMESTAMP
);