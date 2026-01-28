package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
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

func (s *Store) CreateScorebook(ctx context.Context, user User, name, locationText string) (Scorebook, Member, error) {
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
INSERT INTO scorebooks (name, location_text, created_by_user_id, invite_code, updated_at)
VALUES ($1, $2, $3, $4, NOW())
RETURNING id::text, name, location_text, start_time, updated_at, status::text, created_by_user_id, ended_at, invite_code
`, name, locationText, user.ID, invite).Scan(
			&sb.ID,
			&sb.Name,
			&sb.LocationText,
			&sb.StartTime,
			&sb.UpdatedAt,
			&sb.Status,
			&sb.CreatedByUserID,
			&sb.EndedAt,
			&sb.InviteCode,
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
  s.ended_at,
  s.invite_code,
  m.id::text AS my_member_id,
  m.role::text AS my_role,
  (SELECT COUNT(*) FROM scorebook_members mm WHERE mm.scorebook_id = s.id) AS member_count
FROM scorebooks s
JOIN scorebook_members m ON m.scorebook_id = s.id AND m.user_id = $1
ORDER BY s.updated_at DESC
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
  s.created_by_user_id,
  s.ended_at,
  s.invite_code,
  m.id::text AS my_member_id,
  m.role::text AS my_role
FROM scorebooks s
JOIN scorebook_members m ON m.scorebook_id = s.id AND m.user_id = $2
WHERE s.id = $1::uuid
`, scorebookID, userID).Scan(
		&sb.ID,
		&sb.Name,
		&sb.LocationText,
		&sb.StartTime,
		&sb.UpdatedAt,
		&sb.Status,
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
  m.score
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
  AND EXISTS (
    SELECT 1 FROM scorebook_members m
    WHERE m.scorebook_id = s.id AND m.user_id = $2 AND m.role = 'owner'
  )
RETURNING id::text, name, location_text, start_time, updated_at, status::text, created_by_user_id, ended_at, invite_code
`, scorebookID, userID, name).Scan(
		&sb.ID,
		&sb.Name,
		&sb.LocationText,
		&sb.StartTime,
		&sb.UpdatedAt,
		&sb.Status,
		&sb.CreatedByUserID,
		&sb.EndedAt,
		&sb.InviteCode,
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
  AND s.status = 'recording'
  AND EXISTS (
    SELECT 1 FROM scorebook_members m
    WHERE m.scorebook_id = s.id AND m.user_id = $2 AND m.role = 'owner'
  )
RETURNING id::text, name, location_text, start_time, updated_at, status::text, created_by_user_id, ended_at, invite_code
`, scorebookID, userID).Scan(
		&sb.ID,
		&sb.Name,
		&sb.LocationText,
		&sb.StartTime,
		&sb.UpdatedAt,
		&sb.Status,
		&sb.CreatedByUserID,
		&sb.EndedAt,
		&sb.InviteCode,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Scorebook{}, ErrForbidden
		}
		return Scorebook{}, err
	}
	return sb, nil
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
	err = tx.QueryRow(ctx, `SELECT status::text FROM scorebooks WHERE id = $1::uuid`, scorebookID).Scan(&status)
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

func (s *Store) CreateRecord(ctx context.Context, scorebookID string, userID int64, toMemberID string, delta int64, note string) (ScoreRecord, error) {
	if delta <= 0 {
		return ScoreRecord{}, ErrInvalidDelta
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ScoreRecord{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	err = tx.QueryRow(ctx, `SELECT status::text FROM scorebooks WHERE id = $1::uuid FOR UPDATE`, scorebookID).Scan(&status)
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
RETURNING id::text, scorebook_id::text, from_member_id::text, to_member_id::text, delta, note, created_at
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
  m.score
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
SELECT id::text, scorebook_id::text, from_member_id::text, to_member_id::text, delta, note, created_at
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
SELECT id::text, name, status::text, updated_at
FROM scorebooks
WHERE invite_code = $1
`, code).Scan(&info.ScorebookID, &info.Name, &info.Status, &info.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InviteInfo{}, ErrNotFound
		}
		return InviteInfo{}, err
	}
	return info, nil
}

func (s *Store) ScorebookIDByInviteCode(ctx context.Context, code string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id::text FROM scorebooks WHERE invite_code = $1`, code).Scan(&id)
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
	_, _ = s.pool.Exec(ctx, `UPDATE scorebooks SET updated_at = NOW() WHERE id = $1::uuid`, scorebookID)
}

func (s *Store) EnsureMember(ctx context.Context, scorebookID string, userID int64) (Member, error) {
	var m Member
	err := s.pool.QueryRow(ctx, `
SELECT id::text, scorebook_id::text, user_id, role::text, nickname, avatar_url, joined_at, updated_at
FROM scorebook_members
WHERE scorebook_id = $1::uuid AND user_id = $2
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
SELECT EXISTS(SELECT 1 FROM scorebook_members WHERE scorebook_id = $1::uuid AND user_id = $2)
`, scorebookID, userID).Scan(&ok)
	return ok, err
}

func (s *Store) IsOwner(ctx context.Context, scorebookID string, userID int64) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS(SELECT 1 FROM scorebook_members WHERE scorebook_id = $1::uuid AND user_id = $2 AND role = 'owner')
`, scorebookID, userID).Scan(&ok)
	return ok, err
}

func (s *Store) GetScorebookStatus(ctx context.Context, scorebookID string) (string, error) {
	var status string
	err := s.pool.QueryRow(ctx, `SELECT status::text FROM scorebooks WHERE id = $1::uuid`, scorebookID).Scan(&status)
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
SELECT id::text, name, location_text, start_time, updated_at, status::text, created_by_user_id, ended_at, invite_code
FROM scorebooks
WHERE id = $1::uuid
`, scorebookID).Scan(
		&sb.ID,
		&sb.Name,
		&sb.LocationText,
		&sb.StartTime,
		&sb.UpdatedAt,
		&sb.Status,
		&sb.CreatedByUserID,
		&sb.EndedAt,
		&sb.InviteCode,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Scorebook{}, ErrNotFound
		}
		return Scorebook{}, err
	}
	return sb, nil
}

func (s *Store) fmtErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("store: %w", err)
}
