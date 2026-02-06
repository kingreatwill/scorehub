-- Init schema for 得分簿 (ScoreHub)

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
  id                BIGSERIAL PRIMARY KEY,
  wechat_openid     TEXT NOT NULL UNIQUE,
  wechat_nickname   TEXT NOT NULL DEFAULT '',
  wechat_avatar_url TEXT NOT NULL DEFAULT '',
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS scorebooks (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name               TEXT NOT NULL,
  location_text      TEXT NOT NULL DEFAULT '',
  start_time         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  status             TEXT NOT NULL DEFAULT 'recording'
                     CONSTRAINT scorebooks_status_check CHECK (status IN ('recording', 'ended')),
  book_type          TEXT NOT NULL DEFAULT 'scorebook'
                     CONSTRAINT scorebooks_book_type_check CHECK (book_type IN ('scorebook', 'ledger')),
  created_by_user_id BIGINT NOT NULL REFERENCES users(id),
  ended_at           TIMESTAMPTZ NULL,
  invite_code        TEXT NOT NULL UNIQUE
);

ALTER TABLE scorebooks
  ADD COLUMN IF NOT EXISTS book_type TEXT NOT NULL DEFAULT 'scorebook';

CREATE INDEX IF NOT EXISTS idx_scorebooks_updated_at ON scorebooks(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_scorebooks_created_by ON scorebooks(created_by_user_id);

CREATE TABLE IF NOT EXISTS scorebook_members (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  scorebook_id UUID NOT NULL REFERENCES scorebooks(id) ON DELETE CASCADE,
  user_id      BIGINT REFERENCES users(id) ON DELETE CASCADE,
  role         TEXT NOT NULL DEFAULT 'member'
               CONSTRAINT scorebook_members_role_check CHECK (role IN ('owner', 'member')),
  nickname     TEXT NOT NULL,
  avatar_url   TEXT NOT NULL DEFAULT '',
  score        NUMERIC(12,2) NOT NULL DEFAULT 0,
  joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(scorebook_id, user_id)
);

ALTER TABLE scorebook_members
  ADD COLUMN IF NOT EXISTS score NUMERIC(12,2) NOT NULL DEFAULT 0;

ALTER TABLE scorebook_members
  ALTER COLUMN score TYPE NUMERIC(12,2) USING score::numeric(12,2);

ALTER TABLE scorebook_members
  ALTER COLUMN user_id DROP NOT NULL;

CREATE INDEX IF NOT EXISTS idx_members_scorebook ON scorebook_members(scorebook_id);
CREATE INDEX IF NOT EXISTS idx_members_user ON scorebook_members(user_id);

CREATE TABLE IF NOT EXISTS score_records (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  scorebook_id   UUID NOT NULL REFERENCES scorebooks(id) ON DELETE CASCADE,
  from_member_id UUID NOT NULL REFERENCES scorebook_members(id),
  to_member_id   UUID NOT NULL REFERENCES scorebook_members(id),
  delta          NUMERIC(12,2) NOT NULL CHECK (delta <> 0),
  note           TEXT NOT NULL DEFAULT '',
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_records_scorebook_time ON score_records(scorebook_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_records_to_member ON score_records(to_member_id);
CREATE INDEX IF NOT EXISTS idx_records_from_member ON score_records(from_member_id);

ALTER TABLE score_records DROP CONSTRAINT IF EXISTS score_records_delta_check;

ALTER TABLE score_records
  ALTER COLUMN delta TYPE NUMERIC(12,2) USING delta::numeric(12,2);

ALTER TABLE score_records ADD CONSTRAINT score_records_delta_check CHECK (delta <> 0);
