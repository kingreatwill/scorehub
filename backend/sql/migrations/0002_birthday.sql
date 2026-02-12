-- Birthday contacts

CREATE TABLE IF NOT EXISTS birthday_contacts (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name           TEXT NOT NULL,
  gender         TEXT NOT NULL DEFAULT ''
                 CONSTRAINT birthday_gender_check CHECK (gender IN ('男','女','')),
  phone          TEXT NOT NULL DEFAULT '',
  relation       TEXT NOT NULL DEFAULT '',
  note           TEXT NOT NULL DEFAULT '',
  avatar_url     TEXT NOT NULL DEFAULT '',
  solar_birthday DATE NULL,
  lunar_birthday TEXT NOT NULL DEFAULT '',
  primary_type   TEXT NOT NULL DEFAULT 'solar'
                 CONSTRAINT birthday_primary_type_check CHECK (primary_type IN ('solar','lunar')),
  primary_month  SMALLINT NOT NULL DEFAULT 1,
  primary_day    SMALLINT NOT NULL DEFAULT 1,
  primary_year   SMALLINT NOT NULL DEFAULT 0,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_birthday_user ON birthday_contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_birthday_user_md ON birthday_contacts(user_id, primary_month, primary_day);
