CREATE TABLE board_members (
                               id SERIAL PRIMARY KEY,
                               board_id INTEGER REFERENCES boards(id),
                               user_id INTEGER REFERENCES users(id),
                               role TEXT
);