package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
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
