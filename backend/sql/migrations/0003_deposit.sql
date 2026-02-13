-- Deposit book

CREATE TABLE IF NOT EXISTS deposit_accounts (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  bank       TEXT NOT NULL,
  branch     TEXT NOT NULL DEFAULT '',
  account_no TEXT NOT NULL DEFAULT '',
  holder     TEXT NOT NULL DEFAULT '',
  avatar_url TEXT NOT NULL DEFAULT '',
  note       TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ NULL
);

COMMENT ON TABLE deposit_accounts IS '存款账户';
COMMENT ON COLUMN deposit_accounts.id IS '主键';
COMMENT ON COLUMN deposit_accounts.user_id IS '用户ID';
COMMENT ON COLUMN deposit_accounts.bank IS '银行名称';
COMMENT ON COLUMN deposit_accounts.branch IS '支行';
COMMENT ON COLUMN deposit_accounts.account_no IS '账号';
COMMENT ON COLUMN deposit_accounts.holder IS '户名';
COMMENT ON COLUMN deposit_accounts.avatar_url IS '头像URL';
COMMENT ON COLUMN deposit_accounts.note IS '备注';
COMMENT ON COLUMN deposit_accounts.created_at IS '创建时间';
COMMENT ON COLUMN deposit_accounts.updated_at IS '更新时间';
COMMENT ON COLUMN deposit_accounts.deleted_at IS '软删除时间';

CREATE INDEX IF NOT EXISTS idx_deposit_accounts_user ON deposit_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_deposit_accounts_deleted_at ON deposit_accounts(deleted_at);

CREATE TABLE IF NOT EXISTS deposit_records (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  account_id   UUID NOT NULL REFERENCES deposit_accounts(id) ON DELETE CASCADE,
  currency     TEXT NOT NULL DEFAULT 'CNY'
               CONSTRAINT deposit_currency_check CHECK (currency IN ('CNY','USD')),
  amount       NUMERIC(14,2) NOT NULL DEFAULT 0,
  amount_upper TEXT NOT NULL DEFAULT '',
  term_value   INTEGER NOT NULL DEFAULT 0,
  term_unit    TEXT NOT NULL DEFAULT 'year'
               CONSTRAINT deposit_term_unit_check CHECK (term_unit IN ('year','month')),
  rate         NUMERIC(7,4) NOT NULL DEFAULT 0,
  start_date   DATE NOT NULL,
  end_date     DATE NOT NULL,
  interest     NUMERIC(14,2) NOT NULL DEFAULT 0,
  receipt_no   TEXT NOT NULL DEFAULT '',
  status       TEXT NOT NULL DEFAULT '未到期'
               CONSTRAINT deposit_status_check CHECK (status IN ('未到期','已到期','已支取')),
  withdrawn_at DATE NULL,
  tags         TEXT[] NOT NULL DEFAULT '{}',
  attachments  JSONB NOT NULL DEFAULT '[]'::jsonb,
  note         TEXT NOT NULL DEFAULT '',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at   TIMESTAMPTZ NULL
);

COMMENT ON TABLE deposit_records IS '存款记录';
COMMENT ON COLUMN deposit_records.id IS '主键';
COMMENT ON COLUMN deposit_records.user_id IS '用户ID';
COMMENT ON COLUMN deposit_records.account_id IS '账户ID';
COMMENT ON COLUMN deposit_records.currency IS '币种';
COMMENT ON COLUMN deposit_records.amount IS '存款金额';
COMMENT ON COLUMN deposit_records.amount_upper IS '金额大写';
COMMENT ON COLUMN deposit_records.term_value IS '存期数值';
COMMENT ON COLUMN deposit_records.term_unit IS '存期单位';
COMMENT ON COLUMN deposit_records.rate IS '年化利率(%)';
COMMENT ON COLUMN deposit_records.start_date IS '起息日';
COMMENT ON COLUMN deposit_records.end_date IS '到期日';
COMMENT ON COLUMN deposit_records.interest IS '到期利息';
COMMENT ON COLUMN deposit_records.receipt_no IS '单据号';
COMMENT ON COLUMN deposit_records.status IS '状态';
COMMENT ON COLUMN deposit_records.withdrawn_at IS '支取日';
COMMENT ON COLUMN deposit_records.tags IS '标签列表';
COMMENT ON COLUMN deposit_records.attachments IS '附件列表(JSON)';
COMMENT ON COLUMN deposit_records.note IS '备注';
COMMENT ON COLUMN deposit_records.created_at IS '创建时间';
COMMENT ON COLUMN deposit_records.updated_at IS '更新时间';
COMMENT ON COLUMN deposit_records.deleted_at IS '软删除时间';

CREATE INDEX IF NOT EXISTS idx_deposit_records_user ON deposit_records(user_id);
CREATE INDEX IF NOT EXISTS idx_deposit_records_account ON deposit_records(account_id);
CREATE INDEX IF NOT EXISTS idx_deposit_records_user_status_end ON deposit_records(user_id, status, end_date);
CREATE INDEX IF NOT EXISTS idx_deposit_records_tags_gin ON deposit_records USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_deposit_records_deleted_at ON deposit_records(deleted_at);
