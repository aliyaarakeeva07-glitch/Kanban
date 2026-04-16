CREATE TABLE comments (
                          id SERIAL PRIMARY KEY,
                          task_id INTEGER REFERENCES tasks(id),
                          user_id INTEGER REFERENCES users(id),
                          content TEXT,
                          created_at TIMESTAMP,
                          deleted_at TIMESTAMP
);