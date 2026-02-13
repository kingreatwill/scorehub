package store

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) GetLedger(ctx context.Context, ledgerID string) (Scorebook, error) {
	var sb Scorebook
	err := s.pool.QueryRow(ctx, `
SELECT id::text, name, location_text, start_time, updated_at, status::text, book_type, created_by_user_id, ended_at, invite_code, share_disabled
FROM scorebooks
WHERE id = $1::uuid AND book_type = 'ledger' AND deleted_at IS NULL
`, ledgerID).Scan(
		&sb.ID,
		&sb.Name,
		&sb.LocationText,
		&sb.StartTime,
		&sb.UpdatedAt,
		&sb.Status,
		&sb.BookType,
		&sb.CreatedByUserID,
		&sb.EndedAt,
		&sb.InviteCode,
		&sb.ShareDisabled,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Scorebook{}, ErrNotFound
		}
		return Scorebook{}, err
	}
	return sb, nil
}

func (s *Store) CreateLedger(ctx context.Context, user User, name string) (Scorebook, LedgerMember, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Scorebook{}, LedgerMember{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var sb Scorebook
	var member LedgerMember

	for i := 0; i < 5; i++ {
		invite := randomInviteCode(8)
		err = tx.QueryRow(ctx, `
INSERT INTO scorebooks (name, location_text, book_type, created_by_user_id, invite_code, updated_at)
VALUES ($1, '', 'ledger', $2, $3, NOW())
RETURNING id::text, name, location_text, start_time, updated_at, status::text, book_type, created_by_user_id, ended_at, invite_code, share_disabled
`, name, user.ID, invite).Scan(
			&sb.ID,
			&sb.Name,
			&sb.LocationText,
			&sb.StartTime,
			&sb.UpdatedAt,
			&sb.Status,
			&sb.BookType,
			&sb.CreatedByUserID,
			&sb.EndedAt,
			&sb.InviteCode,
			&sb.ShareDisabled,
		)
		if err == nil {
			break
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			continue
		}
		return Scorebook{}, LedgerMember{}, err
	}
	if err != nil {
		return Scorebook{}, LedgerMember{}, err
	}

	nickname := strings.TrimSpace(user.WeChatNickname)
	if nickname == "" {
		nickname = "我"
	}
	avatarURL := strings.TrimSpace(user.WeChatAvatarURL)

	var ownerUserID sql.NullInt64
	err = tx.QueryRow(ctx, `
INSERT INTO scorebook_members (scorebook_id, user_id, role, nickname, avatar_url, remark, updated_at)
VALUES ($1::uuid, $2, 'owner', $3, $4, $5, NOW())
RETURNING id::text, scorebook_id::text, user_id, role, nickname, avatar_url, remark, score::float8, joined_at, updated_at
`, sb.ID, user.ID, nickname, avatarURL, "").Scan(
		&member.ID,
		&member.LedgerID,
		&ownerUserID,
		&member.Role,
		&member.Nickname,
		&member.AvatarURL,
		&member.Remark,
		&member.Score,
		&member.CreatedAt,
		&member.UpdatedAt,
	)
	if err != nil {
		return Scorebook{}, LedgerMember{}, err
	}
	if ownerUserID.Valid {
		member.UserID = &ownerUserID.Int64
	}

	if err := tx.Commit(ctx); err != nil {
		return Scorebook{}, LedgerMember{}, err
	}
	return sb, member, nil
}

func (s *Store) ListLedgersForUser(ctx context.Context, userID int64) ([]LedgerListItem, error) {
	rows, err := s.pool.Query(ctx, `
SELECT
  s.id::text,
  s.name,
  s.start_time,
  s.updated_at,
  s.status::text,
  s.ended_at,
  (SELECT COUNT(*) FROM scorebook_members m WHERE m.scorebook_id = s.id) AS member_count,
  (SELECT COUNT(*) FROM score_records r WHERE r.scorebook_id = s.id) AS record_count
FROM scorebooks s
WHERE s.book_type = 'ledger' AND s.deleted_at IS NULL
  AND (
    s.created_by_user_id = $1
    OR EXISTS (
      SELECT 1
      FROM scorebook_members m
      WHERE m.scorebook_id = s.id AND m.user_id = $1
    )
  )
ORDER BY
  CASE s.status::text
    WHEN 'recording' THEN 0
    WHEN 'ended' THEN 1
    ELSE 2
  END ASC,
  s.updated_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LedgerListItem
	for rows.Next() {
		var it LedgerListItem
		if err := rows.Scan(
			&it.LedgerID,
			&it.Name,
			&it.StartTime,
			&it.UpdatedAt,
			&it.Status,
			&it.EndedAt,
			&it.MemberCount,
			&it.RecordCount,
		); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) GetLedgerDetail(ctx context.Context, ledgerID string) (Scorebook, []LedgerMember, []LedgerRecord, error) {
	var sb Scorebook
	err := s.pool.QueryRow(ctx, `
SELECT id::text, name, location_text, start_time, updated_at, status::text, book_type, created_by_user_id, ended_at, invite_code, share_disabled
FROM scorebooks
WHERE id = $1::uuid AND book_type = 'ledger' AND deleted_at IS NULL
`, ledgerID).Scan(
		&sb.ID,
		&sb.Name,
		&sb.LocationText,
		&sb.StartTime,
		&sb.UpdatedAt,
		&sb.Status,
		&sb.BookType,
		&sb.CreatedByUserID,
		&sb.EndedAt,
		&sb.InviteCode,
		&sb.ShareDisabled,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Scorebook{}, nil, nil, ErrNotFound
		}
		return Scorebook{}, nil, nil, err
	}

	memRows, err := s.pool.Query(ctx, `
SELECT id::text, scorebook_id::text, user_id, role::text, nickname, avatar_url, remark, score::float8, joined_at, updated_at
FROM scorebook_members
WHERE scorebook_id = $1::uuid
ORDER BY joined_at ASC
`, ledgerID)
	if err != nil {
		return Scorebook{}, nil, nil, err
	}
	defer memRows.Close()

	var members []LedgerMember
	ownerID := ""
	for memRows.Next() {
		var m LedgerMember
		var memberUserID sql.NullInt64
		if err := memRows.Scan(
			&m.ID,
			&m.LedgerID,
			&memberUserID,
			&m.Role,
			&m.Nickname,
			&m.AvatarURL,
			&m.Remark,
			&m.Score,
			&m.CreatedAt,
			&m.UpdatedAt,
		); err != nil {
			return Scorebook{}, nil, nil, err
		}
		if memberUserID.Valid {
			m.UserID = &memberUserID.Int64
		}
		if ownerID == "" && m.Role == "owner" {
			ownerID = m.ID
		}
		members = append(members, m)
	}
	if err := memRows.Err(); err != nil {
		return Scorebook{}, nil, nil, err
	}
	if ownerID == "" && len(members) > 0 {
		ownerID = members[0].ID
		members[0].Role = "owner"
	}

	recRows, err := s.pool.Query(ctx, `
SELECT id::text, scorebook_id::text, from_member_id::text, to_member_id::text, delta::float8, note, created_at
FROM score_records
WHERE scorebook_id = $1::uuid
ORDER BY created_at DESC
`, ledgerID)
	if err != nil {
		return Scorebook{}, nil, nil, err
	}
	defer recRows.Close()

	var records []LedgerRecord
	for recRows.Next() {
		var r LedgerRecord
		var delta float64
		if err := recRows.Scan(
			&r.ID,
			&r.LedgerID,
			&r.FromMemberID,
			&r.ToMemberID,
			&delta,
			&r.Note,
			&r.CreatedAt,
		); err != nil {
			return Scorebook{}, nil, nil, err
		}
		if math.Abs(delta) < 1e-9 {
			r.Type = "remark"
			r.MemberID = r.ToMemberID
			r.Amount = 0
			records = append(records, r)
			continue
		}
		if delta < 0 {
			r.Amount = float64(-delta)
		} else {
			r.Amount = float64(delta)
		}
		if ownerID != "" {
			if r.FromMemberID == ownerID {
				r.Type = "expense"
				r.MemberID = r.ToMemberID
			} else if r.ToMemberID == ownerID {
				r.Type = "income"
				r.MemberID = r.FromMemberID
			}
		}
		if r.Type == "" {
			if delta < 0 {
				r.Type = "expense"
			} else {
				r.Type = "income"
			}
			if r.MemberID == "" {
				r.MemberID = r.ToMemberID
			}
		}
		records = append(records, r)
	}
	if err := recRows.Err(); err != nil {
		return Scorebook{}, nil, nil, err
	}

	return sb, members, records, nil
}

func (s *Store) UpdateLedgerName(ctx context.Context, ledgerID string, userID int64, name string) (Scorebook, error) {
	return s.UpdateLedger(ctx, ledgerID, userID, &name, nil)
}

func (s *Store) UpdateLedger(ctx context.Context, ledgerID string, userID int64, name *string, shareDisabled *bool) (Scorebook, error) {
	if name == nil && shareDisabled == nil {
		return Scorebook{}, ErrInvalidArgument
	}
	var nameVal sql.NullString
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" {
			return Scorebook{}, ErrInvalidArgument
		}
		nameVal = sql.NullString{Valid: true, String: n}
	}
	var shareVal sql.NullBool
	if shareDisabled != nil {
		shareVal = sql.NullBool{Valid: true, Bool: *shareDisabled}
	}

	var sb Scorebook
	err := s.pool.QueryRow(ctx, `
UPDATE scorebooks s
SET name = COALESCE(NULLIF($3, ''), name),
    share_disabled = COALESCE($4, share_disabled),
    updated_at = NOW()
WHERE s.id = $1::uuid
  AND s.book_type = 'ledger'
  AND s.created_by_user_id = $2
  AND s.deleted_at IS NULL
RETURNING id::text, name, location_text, start_time, updated_at, status::text, book_type, created_by_user_id, ended_at, invite_code, share_disabled
`, ledgerID, userID, nameVal, shareVal).Scan(
		&sb.ID,
		&sb.Name,
		&sb.LocationText,
		&sb.StartTime,
		&sb.UpdatedAt,
		&sb.Status,
		&sb.BookType,
		&sb.CreatedByUserID,
		&sb.EndedAt,
		&sb.InviteCode,
		&sb.ShareDisabled,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Scorebook{}, ErrForbidden
		}
		return Scorebook{}, err
	}
	return sb, nil
}

func (s *Store) AddLedgerMember(ctx context.Context, ledgerID string, userID int64, nickname, avatarURL, remark string) (LedgerMember, error) {
	if strings.TrimSpace(nickname) == "" {
		return LedgerMember{}, ErrInvalidArgument
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return LedgerMember{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	var ownerID int64
	err = tx.QueryRow(ctx, `
SELECT status::text, created_by_user_id
FROM scorebooks
WHERE id = $1::uuid AND book_type = 'ledger' AND deleted_at IS NULL
`, ledgerID).Scan(&status, &ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LedgerMember{}, ErrNotFound
		}
		return LedgerMember{}, err
	}
	if ownerID != userID {
		return LedgerMember{}, ErrForbidden
	}
	if status != "recording" {
		return LedgerMember{}, ErrScorebookEnded
	}

	var member LedgerMember
	var memberUserID sql.NullInt64
	err = tx.QueryRow(ctx, `
INSERT INTO scorebook_members (scorebook_id, role, nickname, avatar_url, remark, updated_at)
VALUES ($1::uuid, 'member', $2, $3, $4, NOW())
RETURNING id::text, scorebook_id::text, user_id, role, nickname, avatar_url, remark, score::float8, joined_at, updated_at
`, ledgerID, nickname, avatarURL, remark).Scan(
		&member.ID,
		&member.LedgerID,
		&memberUserID,
		&member.Role,
		&member.Nickname,
		&member.AvatarURL,
		&member.Remark,
		&member.Score,
		&member.CreatedAt,
		&member.UpdatedAt,
	)
	if err != nil {
		return LedgerMember{}, err
	}
	if memberUserID.Valid {
		member.UserID = &memberUserID.Int64
	}
	_, _ = tx.Exec(ctx, `UPDATE scorebooks SET updated_at = NOW() WHERE id = $1::uuid`, ledgerID)

	if err := tx.Commit(ctx); err != nil {
		return LedgerMember{}, err
	}
	return member, nil
}

func (s *Store) UpdateLedgerMember(ctx context.Context, ledgerID string, userID int64, memberID string, nickname, avatarURL, remark string) (LedgerMember, error) {
	if strings.TrimSpace(memberID) == "" {
		return LedgerMember{}, ErrInvalidArgument
	}
	if strings.TrimSpace(nickname) == "" {
		return LedgerMember{}, ErrInvalidArgument
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return LedgerMember{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	var ownerID int64
	err = tx.QueryRow(ctx, `
SELECT status::text, created_by_user_id
FROM scorebooks
WHERE id = $1::uuid AND book_type = 'ledger' AND deleted_at IS NULL
`, ledgerID).Scan(&status, &ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LedgerMember{}, ErrNotFound
		}
		return LedgerMember{}, err
	}
	if status != "recording" {
		return LedgerMember{}, ErrScorebookEnded
	}

	var targetUserID sql.NullInt64
	err = tx.QueryRow(ctx, `
SELECT user_id
FROM scorebook_members
WHERE scorebook_id = $1::uuid AND id = $2::uuid
`, ledgerID, memberID).Scan(&targetUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LedgerMember{}, ErrNotFound
		}
		return LedgerMember{}, err
	}
	if ownerID != userID {
		if !targetUserID.Valid || targetUserID.Int64 != userID {
			return LedgerMember{}, ErrForbidden
		}
	}

	var member LedgerMember
	var memberUserID sql.NullInt64
	err = tx.QueryRow(ctx, `
UPDATE scorebook_members
SET nickname = $3, avatar_url = $4, remark = COALESCE($5, remark), updated_at = NOW()
WHERE scorebook_id = $1::uuid AND id = $2::uuid
RETURNING id::text, scorebook_id::text, user_id, role::text, nickname, avatar_url, remark, score::float8, joined_at, updated_at
`, ledgerID, memberID, nickname, avatarURL, remark).Scan(
		&member.ID,
		&member.LedgerID,
		&memberUserID,
		&member.Role,
		&member.Nickname,
		&member.AvatarURL,
		&member.Remark,
		&member.Score,
		&member.CreatedAt,
		&member.UpdatedAt,
	)
	if err != nil {
		return LedgerMember{}, err
	}
	if memberUserID.Valid {
		member.UserID = &memberUserID.Int64
	}
	_, _ = tx.Exec(ctx, `UPDATE scorebooks SET updated_at = NOW() WHERE id = $1::uuid`, ledgerID)

	if err := tx.Commit(ctx); err != nil {
		return LedgerMember{}, err
	}
	return member, nil
}

func (s *Store) BindLedgerMember(ctx context.Context, ledgerID string, userID int64, memberID string, nickname, avatarURL string) (LedgerMember, error) {
	if strings.TrimSpace(memberID) == "" {
		return LedgerMember{}, ErrInvalidArgument
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return LedgerMember{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	var shareDisabled bool
	err = tx.QueryRow(ctx, `
SELECT status::text, share_disabled
FROM scorebooks
WHERE id = $1::uuid AND book_type = 'ledger' AND deleted_at IS NULL
`, ledgerID).Scan(&status, &shareDisabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LedgerMember{}, ErrNotFound
		}
		return LedgerMember{}, err
	}
	if status != "recording" {
		return LedgerMember{}, ErrScorebookEnded
	}
	if shareDisabled {
		return LedgerMember{}, ErrForbidden
	}

	var existingID string
	err = tx.QueryRow(ctx, `
SELECT id::text
FROM scorebook_members
WHERE scorebook_id = $1::uuid AND user_id = $2
`, ledgerID, userID).Scan(&existingID)
	if err == nil && existingID != "" {
		return LedgerMember{}, ErrConflict
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return LedgerMember{}, err
	}

	var member LedgerMember
	var memberUserID sql.NullInt64
	err = tx.QueryRow(ctx, `
SELECT id::text, scorebook_id::text, user_id, role::text, nickname, avatar_url, remark, score::float8, joined_at, updated_at
FROM scorebook_members
WHERE scorebook_id = $1::uuid AND id = $2::uuid
`, ledgerID, memberID).Scan(
		&member.ID,
		&member.LedgerID,
		&memberUserID,
		&member.Role,
		&member.Nickname,
		&member.AvatarURL,
		&member.Remark,
		&member.Score,
		&member.CreatedAt,
		&member.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LedgerMember{}, ErrNotFound
		}
		return LedgerMember{}, err
	}
	if memberUserID.Valid {
		member.UserID = &memberUserID.Int64
		if member.UserID != nil {
			return LedgerMember{}, ErrConflict
		}
	}

	var updated LedgerMember
	var updatedUserID sql.NullInt64
	err = tx.QueryRow(ctx, `
UPDATE scorebook_members
SET user_id = $3, nickname = COALESCE(NULLIF($4, ''), nickname), avatar_url = COALESCE(NULLIF($5, ''), avatar_url), updated_at = NOW()
WHERE scorebook_id = $1::uuid AND id = $2::uuid AND user_id IS NULL
RETURNING id::text, scorebook_id::text, user_id, role::text, nickname, avatar_url, remark, score::float8, joined_at, updated_at
`, ledgerID, memberID, userID, nickname, avatarURL).Scan(
		&updated.ID,
		&updated.LedgerID,
		&updatedUserID,
		&updated.Role,
		&updated.Nickname,
		&updated.AvatarURL,
		&updated.Remark,
		&updated.Score,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LedgerMember{}, ErrConflict
		}
		return LedgerMember{}, err
	}
	if updatedUserID.Valid {
		updated.UserID = &updatedUserID.Int64
	}

	_, _ = tx.Exec(ctx, `UPDATE scorebooks SET updated_at = NOW() WHERE id = $1::uuid`, ledgerID)

	if err := tx.Commit(ctx); err != nil {
		return LedgerMember{}, err
	}
	return updated, nil
}

func (s *Store) AddLedgerRecord(ctx context.Context, ledgerID string, userID int64, memberID string, recordType string, amount float64, note string) (LedgerRecord, error) {
	if strings.TrimSpace(memberID) == "" {
		return LedgerRecord{}, ErrInvalidArgument
	}
	if recordType != "income" && recordType != "expense" {
		return LedgerRecord{}, ErrInvalidArgument
	}
	var ok bool
	amount, ok = normalizeAmount(amount)
	if !ok {
		return LedgerRecord{}, ErrInvalidArgument
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return LedgerRecord{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	var ownerID int64
	err = tx.QueryRow(ctx, `
SELECT status::text, created_by_user_id
FROM scorebooks
WHERE id = $1::uuid AND book_type = 'ledger' AND deleted_at IS NULL
`, ledgerID).Scan(&status, &ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LedgerRecord{}, ErrNotFound
		}
		return LedgerRecord{}, err
	}
	if ownerID != userID {
		return LedgerRecord{}, ErrForbidden
	}
	if status != "recording" {
		return LedgerRecord{}, ErrScorebookEnded
	}

	var ownerMemberID string
	err = tx.QueryRow(ctx, `
SELECT id::text
FROM scorebook_members
WHERE scorebook_id = $1::uuid AND role = 'owner'
`, ledgerID).Scan(&ownerMemberID)
	if err != nil {
		return LedgerRecord{}, err
	}
	if ownerMemberID == "" {
		err = tx.QueryRow(ctx, `
SELECT id::text
FROM scorebook_members
WHERE scorebook_id = $1::uuid
ORDER BY joined_at ASC
LIMIT 1
`, ledgerID).Scan(&ownerMemberID)
		if err != nil {
			return LedgerRecord{}, err
		}
	}
	if ownerMemberID == memberID {
		return LedgerRecord{}, ErrInvalidArgument
	}

	var memberRemark string
	var tmp string
	err = tx.QueryRow(ctx, `
SELECT id::text, remark
FROM scorebook_members
WHERE scorebook_id = $1::uuid AND id = $2::uuid
`, ledgerID, memberID).Scan(&tmp, &memberRemark)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LedgerRecord{}, ErrNotFound
		}
		return LedgerRecord{}, err
	}
	memberRemark = strings.TrimSpace(memberRemark)
	if recordType == "remark" && memberRemark != "" {
		note = memberRemark
	}

	fromMemberID := ownerMemberID
	toMemberID := memberID
	delta := amount
	if recordType == "expense" {
		delta = -amount
		fromMemberID = memberID
		toMemberID = ownerMemberID
	}

	var record LedgerRecord
	err = tx.QueryRow(ctx, `
INSERT INTO score_records (scorebook_id, from_member_id, to_member_id, delta, note)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5)
RETURNING id::text, scorebook_id::text, from_member_id::text, to_member_id::text, delta::float8, note, created_at
`, ledgerID, fromMemberID, toMemberID, delta, note).Scan(
		&record.ID,
		&record.LedgerID,
		&record.FromMemberID,
		&record.ToMemberID,
		&record.Amount,
		&record.Note,
		&record.CreatedAt,
	)
	if err != nil {
		return LedgerRecord{}, err
	}
	record.Type = recordType
	record.MemberID = memberID
	record.Amount = amount

	if _, err := tx.Exec(ctx, `
UPDATE scorebook_members
SET score = score + $1, updated_at = NOW()
WHERE scorebook_id = $2::uuid AND id = $3::uuid
`, amount, ledgerID, toMemberID); err != nil {
		return LedgerRecord{}, err
	}
	if _, err := tx.Exec(ctx, `
UPDATE scorebook_members
SET score = score - $1, updated_at = NOW()
WHERE scorebook_id = $2::uuid AND id = $3::uuid
`, amount, ledgerID, fromMemberID); err != nil {
		return LedgerRecord{}, err
	}
	_, _ = tx.Exec(ctx, `UPDATE scorebooks SET updated_at = NOW() WHERE id = $1::uuid`, ledgerID)

	if err := tx.Commit(ctx); err != nil {
		return LedgerRecord{}, err
	}
	return record, nil
}

func (s *Store) EndLedger(ctx context.Context, ledgerID string, userID int64) (Scorebook, error) {
	var sb Scorebook
	err := s.pool.QueryRow(ctx, `
UPDATE scorebooks s
SET status = 'ended',
    ended_at = NOW(),
    updated_at = NOW()
WHERE s.id = $1::uuid
  AND s.book_type = 'ledger'
  AND s.status = 'recording'
  AND s.created_by_user_id = $2
  AND s.deleted_at IS NULL
RETURNING id::text, name, location_text, start_time, updated_at, status::text, book_type, created_by_user_id, ended_at, invite_code, share_disabled
`, ledgerID, userID).Scan(
		&sb.ID,
		&sb.Name,
		&sb.LocationText,
		&sb.StartTime,
		&sb.UpdatedAt,
		&sb.Status,
		&sb.BookType,
		&sb.CreatedByUserID,
		&sb.EndedAt,
		&sb.InviteCode,
		&sb.ShareDisabled,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Scorebook{}, ErrNotFound
		}
		return Scorebook{}, err
	}
	return sb, nil
}

func (s *Store) DeleteLedger(ctx context.Context, ledgerID string, userID int64) (Scorebook, error) {
	var status string
	var ownerID int64
	var deletedAt sql.NullTime
	err := s.pool.QueryRow(ctx, `
SELECT status::text, created_by_user_id, deleted_at
FROM scorebooks
WHERE id = $1::uuid AND book_type = 'ledger'
`, ledgerID).Scan(&status, &ownerID, &deletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Scorebook{}, ErrNotFound
		}
		return Scorebook{}, err
	}
	if ownerID != userID {
		return Scorebook{}, ErrForbidden
	}
	if status != "ended" {
		return Scorebook{}, ErrScorebookNotEnded
	}
	if deletedAt.Valid {
		return Scorebook{}, ErrNotFound
	}

	var sb Scorebook
	err = s.pool.QueryRow(ctx, `
UPDATE scorebooks
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1::uuid AND book_type = 'ledger' AND status = 'ended' AND deleted_at IS NULL
RETURNING id::text, name, location_text, start_time, updated_at, status::text, book_type, created_by_user_id, ended_at, invite_code, share_disabled
`, ledgerID).Scan(
		&sb.ID,
		&sb.Name,
		&sb.LocationText,
		&sb.StartTime,
		&sb.UpdatedAt,
		&sb.Status,
		&sb.BookType,
		&sb.CreatedByUserID,
		&sb.EndedAt,
		&sb.InviteCode,
		&sb.ShareDisabled,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Scorebook{}, ErrNotFound
		}
		return Scorebook{}, err
	}
	return sb, nil
}
