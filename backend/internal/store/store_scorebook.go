package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) CreateScorebook(ctx context.Context, user User, name, locationText, bookType string) (Scorebook, Member, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Scorebook{}, Member{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var sb Scorebook
	var owner Member

	for i := 0; i < 5; i++ {
		invite := randomInviteCode(8)
		err = tx.QueryRow(ctx, `
INSERT INTO scorebooks (name, location_text, book_type, created_by_user_id, invite_code, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW())
RETURNING id::text, name, location_text, start_time, updated_at, status::text, book_type, created_by_user_id, ended_at, invite_code, share_disabled
`, name, locationText, bookType, user.ID, invite).Scan(
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
		return Scorebook{}, Member{}, err
	}
	if err != nil {
		return Scorebook{}, Member{}, err
	}

	nickname := strings.TrimSpace(user.WeChatNickname)
	if nickname == "" {
		nickname = "我"
	}
	avatarURL := strings.TrimSpace(user.WeChatAvatarURL)

	err = tx.QueryRow(ctx, `
INSERT INTO scorebook_members (scorebook_id, user_id, role, nickname, avatar_url, updated_at)
VALUES ($1::uuid, $2, 'owner', $3, $4, NOW())
RETURNING id::text, scorebook_id::text, user_id, role::text, nickname, avatar_url, joined_at, updated_at
`, sb.ID, user.ID, nickname, avatarURL).Scan(
		&owner.ID,
		&owner.ScorebookID,
		&owner.UserID,
		&owner.Role,
		&owner.Nickname,
		&owner.AvatarURL,
		&owner.JoinedAt,
		&owner.UpdatedAt,
	)
	if err != nil {
		return Scorebook{}, Member{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Scorebook{}, Member{}, err
	}
	return sb, owner, nil
}

func (s *Store) ListScorebooksForUser(ctx context.Context, userID int64, limit, offset int32) ([]ScorebookListItem, error) {
	rows, err := s.pool.Query(ctx, `
SELECT
  s.id::text,
  s.name,
  s.location_text,
  s.start_time,
  s.updated_at,
  s.status::text,
  s.book_type,
  s.ended_at,
  s.invite_code,
  m.id::text AS my_member_id,
  m.role::text AS my_role,
  (SELECT COUNT(*) FROM scorebook_members mm WHERE mm.scorebook_id = s.id) AS member_count
FROM scorebooks s
JOIN scorebook_members m ON m.scorebook_id = s.id AND m.user_id = $1
WHERE s.book_type = 'scorebook' AND s.deleted_at IS NULL
ORDER BY s.updated_at DESC
LIMIT $2 OFFSET $3
`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ScorebookListItem
	for rows.Next() {
		var it ScorebookListItem
		if err := rows.Scan(
			&it.ScorebookID,
			&it.Name,
			&it.LocationText,
			&it.StartTime,
			&it.UpdatedAt,
			&it.Status,
			&it.BookType,
			&it.EndedAt,
			&it.InviteCode,
			&it.MyMemberID,
			&it.MyRole,
			&it.MemberCount,
		); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) GetScorebookDetail(ctx context.Context, scorebookID string, userID int64) (Scorebook, string, string, []MemberWithScore, error) {
	var sb Scorebook
	var myMemberID string
	var myRole string
	err := s.pool.QueryRow(ctx, `
SELECT
  s.id::text,
  s.name,
  s.location_text,
  s.start_time,
  s.updated_at,
  s.status::text,
  s.book_type,
  s.created_by_user_id,
  s.ended_at,
  s.invite_code,
  m.id::text AS my_member_id,
  m.role::text AS my_role
FROM scorebooks s
JOIN scorebook_members m ON m.scorebook_id = s.id AND m.user_id = $2
WHERE s.id = $1::uuid AND s.book_type = 'scorebook' AND s.deleted_at IS NULL
`, scorebookID, userID).Scan(
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
		&myMemberID,
		&myRole,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Scorebook{}, "", "", nil, ErrNotFound
		}
		return Scorebook{}, "", "", nil, err
	}

	rows, err := s.pool.Query(ctx, `
SELECT
  m.id::text,
  m.scorebook_id::text,
  m.user_id,
  m.role::text,
  m.nickname,
  m.avatar_url,
  m.joined_at,
  m.updated_at,
  m.score::float8
FROM scorebook_members m
WHERE m.scorebook_id = $1::uuid
ORDER BY m.joined_at ASC
`, scorebookID)
	if err != nil {
		return Scorebook{}, "", "", nil, err
	}
	defer rows.Close()

	var members []MemberWithScore
	for rows.Next() {
		var m MemberWithScore
		if err := rows.Scan(
			&m.ID,
			&m.ScorebookID,
			&m.UserID,
			&m.Role,
			&m.Nickname,
			&m.AvatarURL,
			&m.JoinedAt,
			&m.UpdatedAt,
			&m.Score,
		); err != nil {
			return Scorebook{}, "", "", nil, err
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return Scorebook{}, "", "", nil, err
	}

	return sb, myMemberID, myRole, members, nil
}

func (s *Store) UpdateScorebookName(ctx context.Context, scorebookID string, userID int64, name string) (Scorebook, error) {
	var sb Scorebook
	err := s.pool.QueryRow(ctx, `
UPDATE scorebooks s
SET name = $3,
    updated_at = NOW()
WHERE s.id = $1::uuid
  AND s.book_type = 'scorebook'
  AND EXISTS (
    SELECT 1 FROM scorebook_members m
    WHERE m.scorebook_id = s.id AND m.user_id = $2 AND m.role = 'owner'
  )
RETURNING id::text, name, location_text, start_time, updated_at, status::text, book_type, created_by_user_id, ended_at, invite_code, share_disabled
`, scorebookID, userID, name).Scan(
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

func (s *Store) EndScorebook(ctx context.Context, scorebookID string, userID int64) (Scorebook, error) {
	var sb Scorebook
	err := s.pool.QueryRow(ctx, `
UPDATE scorebooks s
SET status = 'ended',
    ended_at = NOW(),
    updated_at = NOW()
WHERE s.id = $1::uuid
  AND s.book_type = 'scorebook'
  AND s.status = 'recording'
  AND EXISTS (
    SELECT 1 FROM scorebook_members m
    WHERE m.scorebook_id = s.id AND m.user_id = $2 AND m.role = 'owner'
  )
RETURNING id::text, name, location_text, start_time, updated_at, status::text, book_type, created_by_user_id, ended_at, invite_code, share_disabled
`, scorebookID, userID).Scan(
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

func (s *Store) DeleteScorebook(ctx context.Context, scorebookID string, userID int64) (Scorebook, error) {
	var status string
	var deletedAt sql.NullTime
	err := s.pool.QueryRow(ctx, `
SELECT status::text, deleted_at
FROM scorebooks
WHERE id = $1::uuid AND book_type = 'scorebook'
`, scorebookID).Scan(&status, &deletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Scorebook{}, ErrNotFound
		}
		return Scorebook{}, err
	}

	var isOwner bool
	err = s.pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM scorebook_members m
  WHERE m.scorebook_id = $1::uuid AND m.user_id = $2 AND m.role = 'owner'
)
`, scorebookID, userID).Scan(&isOwner)
	if err != nil {
		return Scorebook{}, err
	}
	if !isOwner {
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
WHERE id = $1::uuid AND book_type = 'scorebook' AND status = 'ended' AND deleted_at IS NULL
RETURNING id::text, name, location_text, start_time, updated_at, status::text, book_type, created_by_user_id, ended_at, invite_code, share_disabled
`, scorebookID).Scan(
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

// AutoEndInactiveScorebooks ends scorebooks that are still "recording" but have had
// no new score record for the given duration (or since creation if there are no records).
//
// It returns the list of scorebooks that were ended by this call.
func (s *Store) AutoEndInactiveScorebooks(ctx context.Context, inactiveFor time.Duration) ([]Scorebook, error) {
	if inactiveFor <= 0 {
		return nil, nil
	}
	threshold := time.Now().Add(-inactiveFor)

	rows, err := s.pool.Query(ctx, `
UPDATE scorebooks s
SET status = 'ended', ended_at = NOW(), updated_at = NOW()
WHERE s.status = 'recording'
  AND s.book_type = 'scorebook'
  AND s.deleted_at IS NULL
  AND COALESCE(
    (
      SELECT r.created_at
      FROM score_records r
      WHERE r.scorebook_id = s.id
      ORDER BY r.created_at DESC
      LIMIT 1
    ),
    s.start_time
  ) < $1
RETURNING id::text, name, location_text, start_time, updated_at, status::text, book_type, created_by_user_id, ended_at, invite_code, share_disabled
`, threshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Scorebook
	for rows.Next() {
		var sb Scorebook
		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}
		out = append(out, sb)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) JoinScorebook(ctx context.Context, scorebookID string, user User, nickname, avatarURL string) (Member, error) {
	if strings.TrimSpace(nickname) == "" {
		nickname = user.WeChatNickname
	}
	if strings.TrimSpace(nickname) == "" {
		nickname = "成员"
	}
	if strings.TrimSpace(avatarURL) == "" {
		avatarURL = user.WeChatAvatarURL
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Member{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 已加入过：允许（哪怕已结束也允许打开详情），保持幂等。
	var existing Member
	err = tx.QueryRow(ctx, `
SELECT id::text, scorebook_id::text, user_id, role::text, nickname, avatar_url, joined_at, updated_at
FROM scorebook_members
WHERE scorebook_id = $1::uuid AND user_id = $2
`, scorebookID, user.ID).Scan(
		&existing.ID,
		&existing.ScorebookID,
		&existing.UserID,
		&existing.Role,
		&existing.Nickname,
		&existing.AvatarURL,
		&existing.JoinedAt,
		&existing.UpdatedAt,
	)
	if err == nil {
		_, _ = tx.Exec(ctx, `UPDATE scorebook_members SET updated_at = NOW() WHERE id = $1::uuid`, existing.ID)
		_, _ = tx.Exec(ctx, `UPDATE scorebooks SET updated_at = NOW() WHERE id = $1::uuid`, scorebookID)
		if err := tx.Commit(ctx); err != nil {
			return Member{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Member{}, err
	}

	// 若已结束，不允许新增。
	var status string
	err = tx.QueryRow(ctx, `SELECT status::text FROM scorebooks WHERE id = $1::uuid AND book_type = 'scorebook' AND deleted_at IS NULL`, scorebookID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Member{}, ErrNotFound
		}
		return Member{}, err
	}
	if status == "ended" {
		return Member{}, ErrScorebookEnded
	}

	var m Member
	err = tx.QueryRow(ctx, `
INSERT INTO scorebook_members (scorebook_id, user_id, role, nickname, avatar_url, updated_at)
VALUES ($1::uuid, $2, 'member', $3, $4, NOW())
RETURNING id::text, scorebook_id::text, user_id, role::text, nickname, avatar_url, joined_at, updated_at
`, scorebookID, user.ID, nickname, avatarURL).Scan(
		&m.ID,
		&m.ScorebookID,
		&m.UserID,
		&m.Role,
		&m.Nickname,
		&m.AvatarURL,
		&m.JoinedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		return Member{}, err
	}
	_, _ = tx.Exec(ctx, `UPDATE scorebooks SET updated_at = NOW() WHERE id = $1::uuid`, scorebookID)

	if err := tx.Commit(ctx); err != nil {
		return Member{}, err
	}
	return m, nil
}

func (s *Store) UpdateMyProfile(ctx context.Context, scorebookID string, userID int64, nickname, avatarURL string) (Member, error) {
	var m Member
	err := s.pool.QueryRow(ctx, `
UPDATE scorebook_members
SET nickname = COALESCE(NULLIF($3, ''), nickname),
    avatar_url = COALESCE(NULLIF($4, ''), avatar_url),
    updated_at = NOW()
WHERE scorebook_id = $1::uuid AND user_id = $2
  AND EXISTS (
    SELECT 1 FROM scorebooks s
    WHERE s.id = $1::uuid AND s.book_type = 'scorebook' AND s.deleted_at IS NULL
  )
RETURNING id::text, scorebook_id::text, user_id, role::text, nickname, avatar_url, joined_at, updated_at
`, scorebookID, userID, nickname, avatarURL).Scan(
		&m.ID,
		&m.ScorebookID,
		&m.UserID,
		&m.Role,
		&m.Nickname,
		&m.AvatarURL,
		&m.JoinedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Member{}, ErrNotFound
		}
		return Member{}, err
	}
	_, _ = s.pool.Exec(ctx, `UPDATE scorebooks SET updated_at = NOW() WHERE id = $1::uuid`, scorebookID)
	return m, nil
}

func (s *Store) CreateRecord(ctx context.Context, scorebookID string, userID int64, toMemberID string, delta float64, note string) (ScoreRecord, error) {
	if strings.TrimSpace(toMemberID) == "" {
		return ScoreRecord{}, ErrInvalidArgument
	}
	var ok bool
	delta, ok = normalizeAmount(delta)
	if !ok {
		return ScoreRecord{}, ErrInvalidDelta
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ScoreRecord{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	err = tx.QueryRow(ctx, `SELECT status::text FROM scorebooks WHERE id = $1::uuid AND book_type = 'scorebook' AND deleted_at IS NULL FOR UPDATE`, scorebookID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ScoreRecord{}, ErrNotFound
		}
		return ScoreRecord{}, err
	}
	if status != "recording" {
		return ScoreRecord{}, ErrScorebookEnded
	}

	var fromMemberID string
	err = tx.QueryRow(ctx, `
SELECT id::text
FROM scorebook_members
WHERE scorebook_id = $1::uuid AND user_id = $2
`, scorebookID, userID).Scan(&fromMemberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ScoreRecord{}, ErrForbidden
		}
		return ScoreRecord{}, err
	}
	if fromMemberID == toMemberID {
		return ScoreRecord{}, ErrInvalidArgument
	}

	var tmp string
	err = tx.QueryRow(ctx, `
SELECT id::text
FROM scorebook_members
WHERE scorebook_id = $1::uuid AND id = $2::uuid
`, scorebookID, toMemberID).Scan(&tmp)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ScoreRecord{}, ErrNotFound
		}
		return ScoreRecord{}, err
	}

	var r ScoreRecord
	err = tx.QueryRow(ctx, `
INSERT INTO score_records (scorebook_id, from_member_id, to_member_id, delta, note)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5)
RETURNING id::text, scorebook_id::text, from_member_id::text, to_member_id::text, delta::float8, note, created_at
`, scorebookID, fromMemberID, toMemberID, delta, note).Scan(
		&r.ID,
		&r.ScorebookID,
		&r.FromMemberID,
		&r.ToMemberID,
		&r.Delta,
		&r.Note,
		&r.CreatedAt,
	)
	if err != nil {
		return ScoreRecord{}, err
	}

	if _, err := tx.Exec(ctx, `
UPDATE scorebook_members
SET score = score + $1, updated_at = NOW()
WHERE scorebook_id = $2::uuid AND id = $3::uuid
`, delta, scorebookID, toMemberID); err != nil {
		return ScoreRecord{}, err
	}
	if _, err := tx.Exec(ctx, `
UPDATE scorebook_members
SET score = score - $1, updated_at = NOW()
WHERE scorebook_id = $2::uuid AND id = $3::uuid
`, delta, scorebookID, fromMemberID); err != nil {
		return ScoreRecord{}, err
	}
	_, _ = tx.Exec(ctx, `UPDATE scorebooks SET updated_at = NOW() WHERE id = $1::uuid`, scorebookID)

	if err := tx.Commit(ctx); err != nil {
		return ScoreRecord{}, err
	}
	return r, nil
}

func (s *Store) GetTopWinners(ctx context.Context, scorebookID string) ([]MemberWithScore, error) {
	rows, err := s.pool.Query(ctx, `
SELECT
  m.id::text,
  m.scorebook_id::text,
  m.user_id,
  m.role::text,
  m.nickname,
  m.avatar_url,
  m.joined_at,
  m.updated_at,
  m.score::float8
FROM scorebook_members m
WHERE m.scorebook_id = $1::uuid
  AND m.score > 0
ORDER BY m.score DESC, m.updated_at ASC
`, scorebookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MemberWithScore
	for rows.Next() {
		var m MemberWithScore
		if err := rows.Scan(
			&m.ID,
			&m.ScorebookID,
			&m.UserID,
			&m.Role,
			&m.Nickname,
			&m.AvatarURL,
			&m.JoinedAt,
			&m.UpdatedAt,
			&m.Score,
		); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ListRecords(ctx context.Context, scorebookID string, userID int64, limit, offset int32) ([]ScoreRecord, error) {
	var active bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM scorebooks WHERE id = $1::uuid AND book_type = 'scorebook' AND deleted_at IS NULL)
`, scorebookID).Scan(&active)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, ErrNotFound
	}

	var ok bool
	err = s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM scorebook_members WHERE scorebook_id = $1::uuid AND user_id = $2)
`, scorebookID, userID).Scan(&ok)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}

	rows, err := s.pool.Query(ctx, `
SELECT id::text, scorebook_id::text, from_member_id::text, to_member_id::text, delta::float8, note, created_at
FROM score_records
WHERE scorebook_id = $1::uuid
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`, scorebookID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ScoreRecord
	for rows.Next() {
		var r ScoreRecord
		if err := rows.Scan(&r.ID, &r.ScorebookID, &r.FromMemberID, &r.ToMemberID, &r.Delta, &r.Note, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) TouchScorebookUpdatedAt(ctx context.Context, scorebookID string) {
	_, _ = s.pool.Exec(ctx, `UPDATE scorebooks SET updated_at = NOW() WHERE id = $1::uuid AND deleted_at IS NULL`, scorebookID)
}

func (s *Store) EnsureMember(ctx context.Context, scorebookID string, userID int64) (Member, error) {
	var m Member
	err := s.pool.QueryRow(ctx, `
SELECT id::text, scorebook_id::text, user_id, role::text, nickname, avatar_url, joined_at, updated_at
FROM scorebook_members
WHERE scorebook_id = $1::uuid AND user_id = $2
  AND EXISTS (
    SELECT 1 FROM scorebooks s
    WHERE s.id = $1::uuid AND s.deleted_at IS NULL
  )
`, scorebookID, userID).Scan(
		&m.ID,
		&m.ScorebookID,
		&m.UserID,
		&m.Role,
		&m.Nickname,
		&m.AvatarURL,
		&m.JoinedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Member{}, ErrForbidden
		}
		return Member{}, err
	}
	return m, nil
}

func (s *Store) IsMember(ctx context.Context, scorebookID string, userID int64) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1
  FROM scorebook_members m
  JOIN scorebooks s ON s.id = m.scorebook_id
  WHERE m.scorebook_id = $1::uuid AND m.user_id = $2 AND s.deleted_at IS NULL
)
`, scorebookID, userID).Scan(&ok)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (s *Store) IsOwner(ctx context.Context, scorebookID string, userID int64) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1
  FROM scorebook_members m
  JOIN scorebooks s ON s.id = m.scorebook_id
  WHERE m.scorebook_id = $1::uuid AND m.user_id = $2 AND m.role = 'owner' AND s.deleted_at IS NULL
)
`, scorebookID, userID).Scan(&ok)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (s *Store) GetScorebookStatus(ctx context.Context, scorebookID string) (string, error) {
	var status string
	err := s.pool.QueryRow(ctx, `SELECT status::text FROM scorebooks WHERE id = $1::uuid AND book_type = 'scorebook' AND deleted_at IS NULL`, scorebookID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return status, nil
}

func (s *Store) GetScorebook(ctx context.Context, scorebookID string) (Scorebook, error) {
	var sb Scorebook
	err := s.pool.QueryRow(ctx, `
SELECT id::text, name, location_text, start_time, updated_at, status::text, book_type, created_by_user_id, ended_at, invite_code, share_disabled
FROM scorebooks
WHERE id = $1::uuid AND book_type = 'scorebook' AND deleted_at IS NULL
`, scorebookID).Scan(
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
