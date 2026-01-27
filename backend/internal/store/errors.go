package store

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrForbidden       = errors.New("forbidden")
	ErrScorebookEnded  = errors.New("scorebook ended")
	ErrInvalidArgument = errors.New("invalid argument")
)
