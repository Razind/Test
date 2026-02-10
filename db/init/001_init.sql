CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  username VARCHAR(100) UNIQUE NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO users (username, email, password_hash)
VALUES
  ('demo_admin', 'admin@test.local', 'test_hash_123'),
  ('demo_user', 'user@test.local', 'test_hash_456')
ON CONFLICT (username) DO NOTHING;
