CREATE TABLE task_labels (
                             id SERIAL PRIMARY KEY,
                             task_id INTEGER REFERENCES tasks(id),
                             label_id INTEGER REFERENCES labels(id),
                             deleted_at TIMESTAMP
);