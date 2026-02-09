package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Store, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("SCOREHUB_DB_DSN is required (set env or create backend/.env)")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) UpsertUserByOpenID(ctx context.Context, openid, nickname, avatarURL string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
INSERT INTO users (wechat_openid, wechat_nickname, wechat_avatar_url, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (wechat_openid)
DO UPDATE SET
  wechat_nickname = COALESCE(NULLIF(EXCLUDED.wechat_nickname, ''), users.wechat_nickname),
  wechat_avatar_url = COALESCE(NULLIF(EXCLUDED.wechat_avatar_url, ''), users.wechat_avatar_url),
  updated_at = NOW()
RETURNING id, wechat_openid, wechat_nickname, wechat_avatar_url, created_at, updated_at
`, openid, nickname, avatarURL).Scan(
		&u.ID,
		&u.WeChatOpenID,
		&u.WeChatNickname,
		&u.WeChatAvatarURL,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	return u, err
}

func (s *Store) UpdateUserProfile(ctx context.Context, userID int64, nickname, avatarURL *string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
UPDATE users
SET wechat_nickname = COALESCE($2, wechat_nickname),
    wechat_avatar_url = COALESCE($3, wechat_avatar_url),
    updated_at = NOW()
WHERE id = $1
RETURNING id, wechat_openid, wechat_nickname, wechat_avatar_url, created_at, updated_at
`, userID, nickname, avatarURL).Scan(
		&u.ID,
		&u.WeChatOpenID,
		&u.WeChatNickname,
		&u.WeChatAvatarURL,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return u, nil
}

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
		nickname = "成员"
	}

	err = tx.QueryRow(ctx, `
INSERT INTO scorebook_members (scorebook_id, user_id, role, nickname, avatar_url, updated_at)
VALUES ($1::uuid, $2, 'owner', $3, $4, NOW())
RETURNING id::text, scorebook_id::text, user_id, role::text, nickname, avatar_url, joined_at, updated_at
`, sb.ID, user.ID, nickname, user.WeChatAvatarURL).Scan(
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

func (s *Store) ListScorebooksForUser(ctx context.Context, userID int64) ([]ScorebookListItem, error) {
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
  s.share_disabled,
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
		&sb.ShareDisabled,
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
SET name = $3, updated_at = NOW()
WHERE s.id = $1::uuid
  AND s.book_type = 'scorebook'
  AND s.deleted_at IS NULL
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
			return Scorebook{}, ErrForbidden
		}
		return Scorebook{}, err
	}
	return sb, nil
}

func (s *Store) EndScorebook(ctx context.Context, scorebookID string, userID int64) (Scorebook, error) {
	var sb Scorebook
	err := s.pool.QueryRow(ctx, `
UPDATE scorebooks s
SET status = 'ended', ended_at = NOW(), updated_at = NOW()
WHERE s.id = $1::uuid
  AND s.book_type = 'scorebook'
  AND s.status = 'recording'
  AND s.deleted_at IS NULL
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
			return Scorebook{}, ErrForbidden
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
	if deletedAt.Valid {
		return Scorebook{}, ErrNotFound
	}

	var isOwner bool
	err = s.pool.QueryRow(ctx, `
SELECT EXISTS(
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

	// 未加入过：仅允许进行中的得分簿加入。
	var status string
	err = tx.QueryRow(ctx, `SELECT status::text FROM scorebooks WHERE id = $1::uuid AND book_type = 'scorebook' AND deleted_at IS NULL`, scorebookID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Member{}, ErrNotFound
		}
		return Member{}, err
	}
	if status != "recording" {
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
SET nickname = $3, avatar_url = $4, updated_at = NOW()
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
ORDER BY m.score DESC, m.joined_at ASC
LIMIT 3
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
	if err := s.pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM scorebooks
  WHERE id = $1::uuid AND book_type = 'scorebook' AND deleted_at IS NULL
)
`, scorebookID).Scan(&active); err != nil {
		return nil, err
	}
	if !active {
		return nil, ErrNotFound
	}

	var ok bool
	err := s.pool.QueryRow(ctx, `
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

func (s *Store) GetInviteInfo(ctx context.Context, code string) (InviteInfo, error) {
	var info InviteInfo
	err := s.pool.QueryRow(ctx, `
SELECT id::text, book_type, name, status::text, share_disabled, updated_at
FROM scorebooks
WHERE invite_code = $1 AND deleted_at IS NULL
`, code).Scan(&info.BookID, &info.BookType, &info.Name, &info.Status, &info.ShareDisabled, &info.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InviteInfo{}, ErrNotFound
		}
		return InviteInfo{}, err
	}
	return info, nil
}

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

func (s *Store) ScorebookIDByInviteCode(ctx context.Context, code string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id::text FROM scorebooks WHERE invite_code = $1 AND book_type = 'scorebook' AND deleted_at IS NULL`, code).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return id, nil
}

func randomInviteCode(n int) string {
	const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	var b strings.Builder
	b.Grow(n)
	max := big.NewInt(int64(len(alphabet)))
	for i := 0; i < n; i++ {
		x, err := rand.Int(rand.Reader, max)
		if err != nil {
			// fallback: time-based
			b.WriteByte(alphabet[time.Now().UnixNano()%int64(len(alphabet))])
			continue
		}
		b.WriteByte(alphabet[x.Int64()])
	}
	return b.String()
}

func (s *Store) GetUserByID(ctx context.Context, userID int64) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
SELECT id, wechat_openid, wechat_nickname, wechat_avatar_url, created_at, updated_at
FROM users
WHERE id = $1
`, userID).Scan(
		&u.ID,
		&u.WeChatOpenID,
		&u.WeChatNickname,
		&u.WeChatAvatarURL,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return u, nil
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
	return ok, err
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
	return ok, err
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
FOR UPDATE
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
	if status == "ended" {
		return LedgerMember{}, ErrScorebookEnded
	}

	var member LedgerMember
	var memberUserID sql.NullInt64
	err = tx.QueryRow(ctx, `
INSERT INTO scorebook_members (scorebook_id, user_id, role, nickname, avatar_url, remark, updated_at)
VALUES ($1::uuid, NULL, 'member', $2, $3, $4, NOW())
RETURNING id::text, scorebook_id::text, user_id, role, nickname, avatar_url, remark, score::float8, joined_at, updated_at
`, ledgerID, strings.TrimSpace(nickname), strings.TrimSpace(avatarURL), strings.TrimSpace(remark)).Scan(
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
FOR UPDATE
`, ledgerID).Scan(&status, &ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LedgerMember{}, ErrNotFound
		}
		return LedgerMember{}, err
	}
	if status == "ended" {
		return LedgerMember{}, ErrScorebookEnded
	}

	var targetUserID sql.NullInt64
	var oldRemark string
	err = tx.QueryRow(ctx, `
SELECT user_id, remark
FROM scorebook_members
WHERE scorebook_id = $1::uuid AND id = $2::uuid
FOR UPDATE
`, ledgerID, memberID).Scan(&targetUserID, &oldRemark)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LedgerMember{}, ErrNotFound
		}
		return LedgerMember{}, err
	}

	isOwner := ownerID == userID
	if !isOwner {
		if !targetUserID.Valid || targetUserID.Int64 != userID {
			return LedgerMember{}, ErrForbidden
		}
	}

	var member LedgerMember
	var memberUserID sql.NullInt64
	if isOwner {
		err = tx.QueryRow(ctx, `
UPDATE scorebook_members
SET nickname = $3, avatar_url = $4, remark = $5, updated_at = NOW()
WHERE scorebook_id = $1::uuid AND id = $2::uuid
RETURNING id::text, scorebook_id::text, user_id, role::text, nickname, avatar_url, remark, score::float8, joined_at, updated_at
`, ledgerID, memberID, strings.TrimSpace(nickname), strings.TrimSpace(avatarURL), strings.TrimSpace(remark)).Scan(
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
	} else {
		err = tx.QueryRow(ctx, `
UPDATE scorebook_members
SET nickname = $3, avatar_url = $4, updated_at = NOW()
WHERE scorebook_id = $1::uuid AND id = $2::uuid
RETURNING id::text, scorebook_id::text, user_id, role::text, nickname, avatar_url, remark, score::float8, joined_at, updated_at
`, ledgerID, memberID, strings.TrimSpace(nickname), strings.TrimSpace(avatarURL)).Scan(
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
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LedgerMember{}, ErrNotFound
		}
		return LedgerMember{}, err
	}
	if memberUserID.Valid {
		member.UserID = &memberUserID.Int64
	}

	if isOwner {
		prev := strings.TrimSpace(oldRemark)
		next := strings.TrimSpace(remark)
		if prev != next {
			note := next
			if note == "" {
				note = "备注已清空"
			}
			var ownerMemberID string
			err = tx.QueryRow(ctx, `
SELECT id::text
FROM scorebook_members
WHERE scorebook_id = $1::uuid AND role = 'owner'
LIMIT 1
`, ledgerID).Scan(&ownerMemberID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					err = tx.QueryRow(ctx, `
SELECT id::text
FROM scorebook_members
WHERE scorebook_id = $1::uuid
ORDER BY joined_at ASC
LIMIT 1
`, ledgerID).Scan(&ownerMemberID)
					if err != nil {
						return LedgerMember{}, err
					}
					_, _ = tx.Exec(ctx, `
UPDATE scorebook_members
SET role = 'owner', updated_at = NOW()
WHERE id = $1::uuid AND role <> 'owner'
`, ownerMemberID)
				} else {
					return LedgerMember{}, err
				}
			}
			_, err = tx.Exec(ctx, `
INSERT INTO score_records (scorebook_id, from_member_id, to_member_id, delta, note)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5)
`, ledgerID, ownerMemberID, memberID, 0, note)
			if err != nil {
				return LedgerMember{}, err
			}
		}
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
FOR UPDATE
`, ledgerID).Scan(&status, &shareDisabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LedgerMember{}, ErrNotFound
		}
		return LedgerMember{}, err
	}
	if status == "ended" {
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
LIMIT 1
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
FOR UPDATE
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
		return LedgerMember{}, ErrConflict
	}

	nickname = strings.TrimSpace(nickname)
	avatarURL = strings.TrimSpace(avatarURL)

	var updated LedgerMember
	var updatedUserID sql.NullInt64
	err = tx.QueryRow(ctx, `
UPDATE scorebook_members
SET user_id = $3,
    nickname = COALESCE(NULLIF($4, ''), nickname),
    avatar_url = COALESCE(NULLIF($5, ''), avatar_url),
    updated_at = NOW()
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
	var ok bool
	amount, ok = normalizeAmount(amount)
	if !ok {
		return LedgerRecord{}, ErrInvalidArgument
	}
	recordType = strings.ToLower(strings.TrimSpace(recordType))
	if recordType != "expense" && recordType != "income" {
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
FOR UPDATE
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
	if status == "ended" {
		return LedgerRecord{}, ErrScorebookEnded
	}

	var ownerMemberID string
	err = tx.QueryRow(ctx, `
SELECT id::text
FROM scorebook_members
WHERE scorebook_id = $1::uuid AND role = 'owner'
LIMIT 1
`, ledgerID).Scan(&ownerMemberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, `
SELECT id::text
FROM scorebook_members
WHERE scorebook_id = $1::uuid
ORDER BY joined_at ASC
LIMIT 1
`, ledgerID).Scan(&ownerMemberID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return LedgerRecord{}, ErrInvalidArgument
				}
				return LedgerRecord{}, err
			}
			_, _ = tx.Exec(ctx, `
UPDATE scorebook_members
SET role = 'owner', updated_at = NOW()
WHERE id = $1::uuid AND role <> 'owner'
`, ownerMemberID)
		} else {
			return LedgerRecord{}, err
		}
	}
	if ownerMemberID == memberID {
		return LedgerRecord{}, ErrInvalidArgument
	}

	var tmp string
	var memberRemark string
	err = tx.QueryRow(ctx, `
SELECT id::text, remark
FROM scorebook_members
WHERE scorebook_id = $1::uuid AND id = $2::uuid
`, ledgerID, memberID).Scan(&tmp, &memberRemark)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LedgerRecord{}, ErrInvalidArgument
		}
		return LedgerRecord{}, err
	}
	note = strings.TrimSpace(note)
	memberRemark = strings.TrimSpace(memberRemark)
	if note == "" {
		note = memberRemark
	}

	fromMemberID := ownerMemberID
	toMemberID := memberID
	if recordType == "income" {
		fromMemberID = memberID
		toMemberID = ownerMemberID
	}

	var record LedgerRecord
	err = tx.QueryRow(ctx, `
INSERT INTO score_records (scorebook_id, from_member_id, to_member_id, delta, note)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5)
RETURNING id::text, scorebook_id::text, from_member_id::text, to_member_id::text, delta::float8, note, created_at
`, ledgerID, fromMemberID, toMemberID, amount, note).Scan(
		&record.ID,
		&record.LedgerID,
		&record.FromMemberID,
		&record.ToMemberID,
		&amount,
		&record.Note,
		&record.CreatedAt,
	)
	if err != nil {
		return LedgerRecord{}, err
	}
	record.Type = recordType
	record.Amount = amount
	record.MemberID = memberID

	_, _ = tx.Exec(ctx, `
UPDATE scorebook_members
SET score = score + $1, updated_at = NOW()
WHERE scorebook_id = $2::uuid AND id = $3::uuid
`, amount, ledgerID, toMemberID)

	_, _ = tx.Exec(ctx, `
UPDATE scorebook_members
SET score = score - $1, updated_at = NOW()
WHERE scorebook_id = $2::uuid AND id = $3::uuid
`, amount, ledgerID, fromMemberID)

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
SET status = 'ended', ended_at = NOW(), updated_at = NOW()
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
			return Scorebook{}, ErrForbidden
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
	if deletedAt.Valid {
		return Scorebook{}, ErrNotFound
	}
	if ownerID != userID {
		return Scorebook{}, ErrForbidden
	}
	if status != "ended" {
		return Scorebook{}, ErrScorebookNotEnded
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

func normalizeAmount(v float64) (float64, bool) {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	rounded := math.Round(v*100) / 100
	if math.Abs(v-rounded) > 1e-6 {
		return 0, false
	}
	return rounded, true
}

func (s *Store) fmtErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("store: %w", err)
}
