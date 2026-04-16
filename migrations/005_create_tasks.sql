CREATE TABLE tasks (
                       id SERIAL PRIMARY KEY,
                       board_id INTEGER REFERENCES boards(id),
                       column_id INTEGER REFERENCES columns(id),
                       title TEXT,
                       description TEXT,
                       priority TEXT CHECK (priority IN ('high','medium','low')),
                       created_by INTEGER REFERENCES users(id),
                       assigned_to INTEGER REFERENCES users(id),
                       position INTEGER,
                       deadline TIMESTAMP,
                       done_at TIMESTAMP,
                       archived_at TIMESTAMP,
                       created_at TIMESTAMP,
                       updated_at TIMESTAMP,
                       deleted_at TIMESTAMP
);