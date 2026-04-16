CREATE TABLE columns (
                         id SERIAL PRIMARY KEY,
                         board_id INTEGER REFERENCES boards(id),
                         title TEXT,
                         position INTEGER,
                         last_position INTEGER,
                         deleted_at TIMESTAMP
);