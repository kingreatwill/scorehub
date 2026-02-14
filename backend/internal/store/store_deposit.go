package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateDepositAccount(ctx context.Context, userID int64, in DepositAccountInput) (DepositAccount, error) {
	bank := strings.TrimSpace(in.Bank)
	if bank == "" {
		return DepositAccount{}, ErrInvalidArgument
	}

	var account DepositAccount
	err := s.pool.QueryRow(ctx, `
INSERT INTO deposit_accounts
  (user_id, bank, branch, account_no, holder, avatar_url, note, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
RETURNING id::text, user_id, bank, branch, account_no, holder, avatar_url, note,
          created_at, updated_at, deleted_at
`, userID, bank, in.Branch, in.AccountNo, in.Holder, in.AvatarURL, in.Note).Scan(
		&account.ID,
		&account.UserID,
		&account.Bank,
		&account.Branch,
		&account.AccountNo,
		&account.Holder,
		&account.AvatarURL,
		&account.Note,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.DeletedAt,
	)
	if err != nil {
		return DepositAccount{}, err
	}
	return account, nil
}

func (s *Store) ListDepositAccounts(ctx context.Context, userID int64, limit, offset int32) ([]DepositAccount, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id::text, user_id, bank, branch, account_no, holder, avatar_url, note,
       created_at, updated_at, deleted_at
FROM deposit_accounts
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DepositAccount
	for rows.Next() {
		var item DepositAccount
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Bank,
			&item.Branch,
			&item.AccountNo,
			&item.Holder,
			&item.AvatarURL,
			&item.Note,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.DeletedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) GetDepositAccount(ctx context.Context, userID int64, id string) (DepositAccount, error) {
	var account DepositAccount
	err := s.pool.QueryRow(ctx, `
SELECT id::text, user_id, bank, branch, account_no, holder, avatar_url, note,
       created_at, updated_at, deleted_at
FROM deposit_accounts
WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
`, id, userID).Scan(
		&account.ID,
		&account.UserID,
		&account.Bank,
		&account.Branch,
		&account.AccountNo,
		&account.Holder,
		&account.AvatarURL,
		&account.Note,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DepositAccount{}, ErrNotFound
		}
		return DepositAccount{}, err
	}
	return account, nil
}

func (s *Store) UpdateDepositAccount(ctx context.Context, userID int64, id string, in DepositAccountUpdate) (DepositAccount, error) {
	hasUpdate := false
	if in.Bank != nil || in.Branch != nil || in.AccountNo != nil || in.Holder != nil || in.AvatarURL != nil || in.Note != nil {
		hasUpdate = true
	}
	if !hasUpdate {
		return DepositAccount{}, ErrInvalidArgument
	}

	var bank sql.NullString
	if in.Bank != nil {
		val := strings.TrimSpace(*in.Bank)
		if val == "" {
			return DepositAccount{}, ErrInvalidArgument
		}
		bank = sql.NullString{Valid: true, String: val}
	}
	var branch sql.NullString
	if in.Branch != nil {
		branch = sql.NullString{Valid: true, String: strings.TrimSpace(*in.Branch)}
	}
	var accountNo sql.NullString
	if in.AccountNo != nil {
		accountNo = sql.NullString{Valid: true, String: strings.TrimSpace(*in.AccountNo)}
	}
	var holder sql.NullString
	if in.Holder != nil {
		holder = sql.NullString{Valid: true, String: strings.TrimSpace(*in.Holder)}
	}
	var avatar sql.NullString
	if in.AvatarURL != nil {
		avatar = sql.NullString{Valid: true, String: strings.TrimSpace(*in.AvatarURL)}
	}
	var note sql.NullString
	if in.Note != nil {
		note = sql.NullString{Valid: true, String: strings.TrimSpace(*in.Note)}
	}

	var account DepositAccount
	err := s.pool.QueryRow(ctx, `
UPDATE deposit_accounts
SET bank = COALESCE($3, bank),
    branch = COALESCE($4, branch),
    account_no = COALESCE($5, account_no),
    holder = COALESCE($6, holder),
    avatar_url = COALESCE($7, avatar_url),
    note = COALESCE($8, note),
    updated_at = NOW()
WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
RETURNING id::text, user_id, bank, branch, account_no, holder, avatar_url, note,
          created_at, updated_at, deleted_at
`, id, userID, bank, branch, accountNo, holder, avatar, note).Scan(
		&account.ID,
		&account.UserID,
		&account.Bank,
		&account.Branch,
		&account.AccountNo,
		&account.Holder,
		&account.AvatarURL,
		&account.Note,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DepositAccount{}, ErrNotFound
		}
		return DepositAccount{}, err
	}
	return account, nil
}

func (s *Store) DeleteDepositAccount(ctx context.Context, userID int64, id string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	tag, err := tx.Exec(ctx, `
UPDATE deposit_accounts
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	_, err = tx.Exec(ctx, `
UPDATE deposit_records
SET deleted_at = NOW(), updated_at = NOW()
WHERE account_id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
`, id, userID)
	if err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Store) CreateDepositRecord(ctx context.Context, userID int64, in DepositRecordInput) (DepositRecord, error) {
	if strings.TrimSpace(in.AccountID) == "" {
		return DepositRecord{}, ErrInvalidArgument
	}

	var exists bool
	if err := s.pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM deposit_accounts
  WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
)
`, in.AccountID, userID).Scan(&exists); err != nil {
		return DepositRecord{}, err
	}
	if !exists {
		return DepositRecord{}, ErrNotFound
	}

	attachmentsJSON, err := json.Marshal(in.Attachments)
	if err != nil {
		return DepositRecord{}, ErrInvalidArgument
	}
	attachments := string(attachmentsJSON)
	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}

	var record DepositRecord
	var withdrawn sql.NullTime
	if in.WithdrawnAt != nil {
		withdrawn = sql.NullTime{Valid: true, Time: *in.WithdrawnAt}
	}
	var attachmentsOut []byte
	err = s.pool.QueryRow(ctx, `
INSERT INTO deposit_records
  (user_id, account_id, currency, amount, amount_upper, term_value, term_unit, rate,
   start_date, end_date, interest, receipt_no, status, withdrawn_at, tags, attachments, note, updated_at)
VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $8,
        $9, $10, $11, $12, $13, $14, $15, $16::jsonb, $17, NOW())
RETURNING id::text, user_id, account_id, currency, amount, amount_upper, term_value, term_unit, rate,
          start_date, end_date, interest, receipt_no, status, withdrawn_at, tags, attachments, note,
          created_at, updated_at, deleted_at
`, userID, in.AccountID, in.Currency, in.Amount, in.AmountUpper, in.TermValue, in.TermUnit, in.Rate,
		in.StartDate, in.EndDate, in.Interest, in.ReceiptNo, in.Status, withdrawn, tags, attachments, in.Note).Scan(
		&record.ID,
		&record.UserID,
		&record.AccountID,
		&record.Currency,
		&record.Amount,
		&record.AmountUpper,
		&record.TermValue,
		&record.TermUnit,
		&record.Rate,
		&record.StartDate,
		&record.EndDate,
		&record.Interest,
		&record.ReceiptNo,
		&record.Status,
		&withdrawn,
		&record.Tags,
		&attachmentsOut,
		&record.Note,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.DeletedAt,
	)
	if err != nil {
		return DepositRecord{}, err
	}
	if withdrawn.Valid {
		t := withdrawn.Time
		record.WithdrawnAt = &t
	}
	if len(attachmentsOut) > 0 {
		_ = json.Unmarshal(attachmentsOut, &record.Attachments)
	}
	return record, nil
}

func (s *Store) ListDepositRecords(ctx context.Context, userID int64, accountID string, status string, tags []string, limit, offset int32) ([]DepositRecord, error) {
	var rows pgx.Rows
	var err error
	if accountID != "" {
		rows, err = s.pool.Query(ctx, `
SELECT id::text, user_id, account_id::text, currency, amount, amount_upper, term_value, term_unit, rate,
       start_date, end_date, interest, receipt_no, status, withdrawn_at, tags, attachments, note,
       created_at, updated_at, deleted_at
FROM deposit_records
WHERE user_id = $1 AND account_id = $2::uuid AND deleted_at IS NULL
  AND ($3 = '' OR status = $3)
  AND (array_length($4::text[], 1) IS NULL OR tags && $4::text[])
ORDER BY
  CASE status WHEN '未到期' THEN 0 WHEN '已到期' THEN 1 ELSE 2 END,
  COALESCE(withdrawn_at, end_date) ASC
LIMIT $5 OFFSET $6
`, userID, accountID, status, tags, limit, offset)
	} else {
		rows, err = s.pool.Query(ctx, `
SELECT id::text, user_id, account_id::text, currency, amount, amount_upper, term_value, term_unit, rate,
       start_date, end_date, interest, receipt_no, status, withdrawn_at, tags, attachments, note,
       created_at, updated_at, deleted_at
FROM deposit_records
WHERE user_id = $1 AND deleted_at IS NULL
  AND ($2 = '' OR status = $2)
  AND (array_length($3::text[], 1) IS NULL OR tags && $3::text[])
ORDER BY
  CASE status WHEN '未到期' THEN 0 WHEN '已到期' THEN 1 ELSE 2 END,
  COALESCE(withdrawn_at, end_date) ASC
LIMIT $4 OFFSET $5
`, userID, status, tags, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DepositRecord
	for rows.Next() {
		item, err := scanDepositRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) GetDepositRecord(ctx context.Context, userID int64, id string) (DepositRecord, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id::text, user_id, account_id::text, currency, amount, amount_upper, term_value, term_unit, rate,
       start_date, end_date, interest, receipt_no, status, withdrawn_at, tags, attachments, note,
       created_at, updated_at, deleted_at
FROM deposit_records
WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
`, id, userID)
	item, err := scanDepositRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DepositRecord{}, ErrNotFound
		}
		return DepositRecord{}, err
	}
	return item, nil
}

func (s *Store) UpdateDepositRecord(ctx context.Context, userID int64, id string, in DepositRecordUpdate) (DepositRecord, error) {
	hasUpdate := false
	if in.Currency != nil || in.Amount != nil || in.AmountUpper != nil || in.TermValue != nil || in.TermUnit != nil ||
		in.Rate != nil || in.StartDate != nil || in.EndDate != nil || in.Interest != nil || in.ReceiptNo != nil ||
		in.Status != nil || in.WithdrawnAt != nil || in.WithdrawnSetNull || in.Tags != nil || in.Attachments != nil || in.Note != nil {
		hasUpdate = true
	}
	if !hasUpdate {
		return DepositRecord{}, ErrInvalidArgument
	}

	var currency sql.NullString
	if in.Currency != nil {
		currency = sql.NullString{Valid: true, String: strings.TrimSpace(*in.Currency)}
	}
	var amount sql.NullFloat64
	if in.Amount != nil {
		amount = sql.NullFloat64{Valid: true, Float64: *in.Amount}
	}
	var amountUpper sql.NullString
	if in.AmountUpper != nil {
		amountUpper = sql.NullString{Valid: true, String: strings.TrimSpace(*in.AmountUpper)}
	}
	var termValue sql.NullInt64
	if in.TermValue != nil {
		termValue = sql.NullInt64{Valid: true, Int64: int64(*in.TermValue)}
	}
	var termUnit sql.NullString
	if in.TermUnit != nil {
		termUnit = sql.NullString{Valid: true, String: strings.TrimSpace(*in.TermUnit)}
	}
	var rate sql.NullFloat64
	if in.Rate != nil {
		rate = sql.NullFloat64{Valid: true, Float64: *in.Rate}
	}
	var startDate sql.NullTime
	if in.StartDate != nil {
		startDate = sql.NullTime{Valid: true, Time: *in.StartDate}
	}
	var endDate sql.NullTime
	if in.EndDate != nil {
		endDate = sql.NullTime{Valid: true, Time: *in.EndDate}
	}
	var interest sql.NullFloat64
	if in.Interest != nil {
		interest = sql.NullFloat64{Valid: true, Float64: *in.Interest}
	}
	var receiptNo sql.NullString
	if in.ReceiptNo != nil {
		receiptNo = sql.NullString{Valid: true, String: strings.TrimSpace(*in.ReceiptNo)}
	}
	var status sql.NullString
	if in.Status != nil {
		status = sql.NullString{Valid: true, String: strings.TrimSpace(*in.Status)}
	}
	var withdrawn sql.NullTime
	if in.WithdrawnAt != nil {
		withdrawn = sql.NullTime{Valid: true, Time: *in.WithdrawnAt}
	}

	var tagsArray any
	if in.Tags != nil {
		tagsArray = *in.Tags
	}

	var attachments any
	if in.Attachments != nil {
		attachmentsJSON, err := json.Marshal(*in.Attachments)
		if err != nil {
			return DepositRecord{}, ErrInvalidArgument
		}
		attachments = string(attachmentsJSON)
	}

	var note sql.NullString
	if in.Note != nil {
		note = sql.NullString{Valid: true, String: strings.TrimSpace(*in.Note)}
	}

	var record DepositRecord
	var withdrawnOut sql.NullTime
	var attachmentsOut []byte
	err := s.pool.QueryRow(ctx, `
UPDATE deposit_records
SET currency = COALESCE($3, currency),
    amount = COALESCE($4, amount),
    amount_upper = COALESCE($5, amount_upper),
    term_value = COALESCE($6, term_value),
    term_unit = COALESCE($7, term_unit),
    rate = COALESCE($8, rate),
    start_date = COALESCE($9, start_date),
    end_date = COALESCE($10, end_date),
    interest = COALESCE($11, interest),
    receipt_no = COALESCE($12, receipt_no),
    status = COALESCE($13, status),
    withdrawn_at = CASE WHEN $14 THEN NULL ELSE COALESCE($15, withdrawn_at) END,
    tags = COALESCE($16::text[], tags),
    attachments = COALESCE($17::jsonb, attachments),
    note = COALESCE($18, note),
    updated_at = NOW()
WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
RETURNING id::text, user_id, account_id::text, currency, amount, amount_upper, term_value, term_unit, rate,
          start_date, end_date, interest, receipt_no, status, withdrawn_at, tags, attachments, note,
          created_at, updated_at, deleted_at
`, id, userID,
		currency, amount, amountUpper, termValue, termUnit, rate, startDate, endDate,
		interest, receiptNo, status,
		in.WithdrawnSetNull, withdrawn,
		tagsArray, attachments, note).Scan(
		&record.ID,
		&record.UserID,
		&record.AccountID,
		&record.Currency,
		&record.Amount,
		&record.AmountUpper,
		&record.TermValue,
		&record.TermUnit,
		&record.Rate,
		&record.StartDate,
		&record.EndDate,
		&record.Interest,
		&record.ReceiptNo,
		&record.Status,
		&withdrawnOut,
		&record.Tags,
		&attachmentsOut,
		&record.Note,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DepositRecord{}, ErrNotFound
		}
		return DepositRecord{}, err
	}
	if withdrawnOut.Valid {
		t := withdrawnOut.Time
		record.WithdrawnAt = &t
	}
	if len(attachmentsOut) > 0 {
		_ = json.Unmarshal(attachmentsOut, &record.Attachments)
	}
	return record, nil
}

func (s *Store) DeleteDepositRecord(ctx context.Context, userID int64, id string) error {
	tag, err := s.pool.Exec(ctx, `
UPDATE deposit_records
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListDepositTags(ctx context.Context, userID int64, accountID string, status string) ([]DepositTagCount, error) {
	rows, err := s.pool.Query(ctx, `
SELECT tag, COUNT(*)::int
FROM (
  SELECT unnest(tags) AS tag
  FROM deposit_records
  WHERE user_id = $1 AND deleted_at IS NULL
    AND ($2 = '' OR account_id = NULLIF($2,'')::uuid)
    AND ($3 = '' OR status = $3)
) t
WHERE tag <> ''
GROUP BY tag
ORDER BY COUNT(*) DESC, tag ASC
`, userID, accountID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DepositTagCount
	for rows.Next() {
		var item DepositTagCount
		if err := rows.Scan(&item.Tag, &item.Count); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) GetDepositStats(ctx context.Context, userID int64, accountID string, status string, tags []string) (DepositStats, error) {
	stats := DepositStats{}

	rows, err := s.pool.Query(ctx, `
WITH base AS (
  SELECT account_id, currency, amount, rate, tags
  FROM deposit_records
  WHERE user_id = $1 AND deleted_at IS NULL
    AND ($2 = '' OR account_id = NULLIF($2,'')::uuid)
    AND ($3 = '' OR status = $3)
    AND (array_length($4::text[], 1) IS NULL OR tags && $4::text[])
)
SELECT currency, COALESCE(SUM(amount), 0)::float8
FROM base
GROUP BY currency
`, userID, accountID, status, tags)
	if err != nil {
		return stats, err
	}
	for rows.Next() {
		var item DepositCurrencyStat
		if err := rows.Scan(&item.Currency, &item.Amount); err != nil {
			rows.Close()
			return stats, err
		}
		stats.Totals = append(stats.Totals, item)
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
WITH base AS (
  SELECT currency, interest, end_date, tags
  FROM deposit_records
  WHERE user_id = $1 AND deleted_at IS NULL
    AND ($2 = '' OR account_id = NULLIF($2,'')::uuid)
    AND ($3 = '' OR status = $3)
    AND (array_length($4::text[], 1) IS NULL OR tags && $4::text[])
)
SELECT currency, COALESCE(SUM(interest), 0)::float8
FROM base
WHERE date_part('year', end_date) = date_part('year', CURRENT_DATE)
GROUP BY currency
`, userID, accountID, status, tags)
	if err != nil {
		return stats, err
	}
	for rows.Next() {
		var item DepositCurrencyStat
		if err := rows.Scan(&item.Currency, &item.Amount); err != nil {
			rows.Close()
			return stats, err
		}
		stats.AnnualYields = append(stats.AnnualYields, item)
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
WITH base AS (
  SELECT account_id, currency, amount, tags
  FROM deposit_records
  WHERE user_id = $1 AND deleted_at IS NULL
    AND ($2 = '' OR account_id = NULLIF($2,'')::uuid)
    AND ($3 = '' OR status = $3)
    AND (array_length($4::text[], 1) IS NULL OR tags && $4::text[])
)
SELECT account_id::text, currency, COALESCE(SUM(amount), 0)::float8
FROM base
GROUP BY account_id, currency
`, userID, accountID, status, tags)
	if err != nil {
		return stats, err
	}
	for rows.Next() {
		var item DepositAccountStat
		if err := rows.Scan(&item.AccountID, &item.Currency, &item.Amount); err != nil {
			rows.Close()
			return stats, err
		}
		stats.AccountTotals = append(stats.AccountTotals, item)
	}
	rows.Close()

	return stats, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDepositRecord(row rowScanner) (DepositRecord, error) {
	var record DepositRecord
	var withdrawn sql.NullTime
	var attachmentsRaw []byte
	err := row.Scan(
		&record.ID,
		&record.UserID,
		&record.AccountID,
		&record.Currency,
		&record.Amount,
		&record.AmountUpper,
		&record.TermValue,
		&record.TermUnit,
		&record.Rate,
		&record.StartDate,
		&record.EndDate,
		&record.Interest,
		&record.ReceiptNo,
		&record.Status,
		&withdrawn,
		&record.Tags,
		&attachmentsRaw,
		&record.Note,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.DeletedAt,
	)
	if err != nil {
		return DepositRecord{}, err
	}
	if withdrawn.Valid {
		t := withdrawn.Time
		record.WithdrawnAt = &t
	}
	if len(attachmentsRaw) > 0 {
		_ = json.Unmarshal(attachmentsRaw, &record.Attachments)
	}
	return record, nil
}
