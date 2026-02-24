-- Auth enhancements: username/password + optional wechat binding

ALTER TABLE users
  ALTER COLUMN wechat_openid DROP NOT NULL;

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS username TEXT NULL UNIQUE;

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS password_hash TEXT NULL;

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS password_set_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

