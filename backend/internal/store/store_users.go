package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

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
RETURNING id, wechat_openid, wechat_nickname, wechat_avatar_url, username, password_hash, password_set_at, created_at, updated_at
`, openid, nickname, avatarURL).Scan(
		&u.ID,
		&u.WeChatOpenID,
		&u.WeChatNickname,
		&u.WeChatAvatarURL,
		&u.Username,
		&u.PasswordHash,
		&u.PasswordSetAt,
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
RETURNING id, wechat_openid, wechat_nickname, wechat_avatar_url, username, password_hash, password_set_at, created_at, updated_at
`, userID, nickname, avatarURL).Scan(
		&u.ID,
		&u.WeChatOpenID,
		&u.WeChatNickname,
		&u.WeChatAvatarURL,
		&u.Username,
		&u.PasswordHash,
		&u.PasswordSetAt,
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

func (s *Store) GetUserByID(ctx context.Context, userID int64) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
SELECT id, wechat_openid, wechat_nickname, wechat_avatar_url, username, password_hash, password_set_at, created_at, updated_at
FROM users
WHERE id = $1
`, userID).Scan(
		&u.ID,
		&u.WeChatOpenID,
		&u.WeChatNickname,
		&u.WeChatAvatarURL,
		&u.Username,
		&u.PasswordHash,
		&u.PasswordSetAt,
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

func (s *Store) GetUserByUsername(ctx context.Context, username string) (User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return User{}, ErrInvalidArgument
	}
	var u User
	err := s.pool.QueryRow(ctx, `
SELECT id, wechat_openid, wechat_nickname, wechat_avatar_url, username, password_hash, password_set_at, created_at, updated_at
FROM users
WHERE username = $1
`, username).Scan(
		&u.ID,
		&u.WeChatOpenID,
		&u.WeChatNickname,
		&u.WeChatAvatarURL,
		&u.Username,
		&u.PasswordHash,
		&u.PasswordSetAt,
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

func (s *Store) CreateUserWithCredentials(ctx context.Context, username string, passwordHash string) (User, error) {
	username = strings.TrimSpace(username)
	passwordHash = strings.TrimSpace(passwordHash)
	if username == "" || passwordHash == "" {
		return User{}, ErrInvalidArgument
	}
	var u User
	err := s.pool.QueryRow(ctx, `
INSERT INTO users (username, password_hash, password_set_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
RETURNING id, wechat_openid, wechat_nickname, wechat_avatar_url, username, password_hash, password_set_at, created_at, updated_at
`, username, passwordHash).Scan(
		&u.ID,
		&u.WeChatOpenID,
		&u.WeChatNickname,
		&u.WeChatAvatarURL,
		&u.Username,
		&u.PasswordHash,
		&u.PasswordSetAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrConflict
		}
		return User{}, err
	}
	return u, nil
}

// SetUserCredentials sets username/password for the first time.
func (s *Store) SetUserCredentials(ctx context.Context, userID int64, username string, passwordHash string, now time.Time) (User, error) {
	username = strings.TrimSpace(username)
	passwordHash = strings.TrimSpace(passwordHash)
	if userID <= 0 || username == "" || passwordHash == "" {
		return User{}, ErrInvalidArgument
	}
	if now.IsZero() {
		now = time.Now()
	}
	var u User
	err := s.pool.QueryRow(ctx, `
UPDATE users
SET username = $2,
    password_hash = $3,
    password_set_at = $4,
    updated_at = NOW()
WHERE id = $1
  AND (username IS NULL OR username = '')
  AND (password_hash IS NULL OR password_hash = '')
RETURNING id, wechat_openid, wechat_nickname, wechat_avatar_url, username, password_hash, password_set_at, created_at, updated_at
`, userID, username, passwordHash, now).Scan(
		&u.ID,
		&u.WeChatOpenID,
		&u.WeChatNickname,
		&u.WeChatAvatarURL,
		&u.Username,
		&u.PasswordHash,
		&u.PasswordSetAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// user not found or already set
			exists, e2 := s.userExists(ctx, userID)
			if e2 == nil && !exists {
				return User{}, ErrNotFound
			}
			return User{}, ErrConflict
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrConflict
		}
		return User{}, err
	}
	return u, nil
}

func (s *Store) ChangePassword(ctx context.Context, userID int64, passwordHash string) (User, error) {
	passwordHash = strings.TrimSpace(passwordHash)
	if userID <= 0 || passwordHash == "" {
		return User{}, ErrInvalidArgument
	}
	var u User
	err := s.pool.QueryRow(ctx, `
UPDATE users
SET password_hash = $2,
    password_set_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING id, wechat_openid, wechat_nickname, wechat_avatar_url, username, password_hash, password_set_at, created_at, updated_at
`, userID, passwordHash).Scan(
		&u.ID,
		&u.WeChatOpenID,
		&u.WeChatNickname,
		&u.WeChatAvatarURL,
		&u.Username,
		&u.PasswordHash,
		&u.PasswordSetAt,
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

func (s *Store) BindWeChatOpenID(ctx context.Context, userID int64, openid string) (User, error) {
	openid = strings.TrimSpace(openid)
	if userID <= 0 || openid == "" {
		return User{}, ErrInvalidArgument
	}
	var u User
	err := s.pool.QueryRow(ctx, `
UPDATE users
SET wechat_openid = $2,
    updated_at = NOW()
WHERE id = $1
  AND (wechat_openid IS NULL OR wechat_openid = '')
RETURNING id, wechat_openid, wechat_nickname, wechat_avatar_url, username, password_hash, password_set_at, created_at, updated_at
`, userID, openid).Scan(
		&u.ID,
		&u.WeChatOpenID,
		&u.WeChatNickname,
		&u.WeChatAvatarURL,
		&u.Username,
		&u.PasswordHash,
		&u.PasswordSetAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// user not found or already bound
			exists, e2 := s.userExists(ctx, userID)
			if e2 == nil && !exists {
				return User{}, ErrNotFound
			}
			return User{}, ErrConflict
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrConflict
		}
		return User{}, err
	}
	return u, nil
}

func (s *Store) userExists(ctx context.Context, userID int64) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&ok)
	return ok, err
}
