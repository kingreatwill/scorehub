package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

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
