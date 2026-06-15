CREATE TABLE users (
  id     INTEGER PRIMARY KEY,
  name   TEXT NOT NULL,
  email  TEXT NOT NULL,
  active INTEGER NOT NULL DEFAULT 1
);

INSERT INTO users (id, name, email, active) VALUES
  (1, 'Alice',   'alice@example.com',   1),
  (2, 'Bob',     'bob@example.com',     1),
  (3, 'Charlie', 'charlie@example.com', 0);
