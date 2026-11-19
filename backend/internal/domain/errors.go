package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("email already taken")
	ErrSlugTaken          = errors.New("status slug already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrLimitReached       = errors.New("limit reached")
	ErrProviderDisabled   = errors.New("provider disabled")
)

type ValidationError struct {
	Field   string
	Message string
}

func (err ValidationError) Error() string {
	return err.Field + ": " + err.Message
}
