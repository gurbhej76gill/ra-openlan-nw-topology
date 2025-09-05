package apperrors

import "errors"

var (
	ErrBadRequest   = errors.New("bad_request")
	ErrUnauthorized = errors.New("unauthorized")
	ErrNotFound     = errors.New("not_found")
	ErrInternal     = errors.New("internal_error")
)
