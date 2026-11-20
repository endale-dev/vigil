package errors

import stderrors "errors"

var (
	ErrNotFound           = stderrors.New("not found")
	ErrEmailTaken         = stderrors.New("email already taken")
	ErrSlugTaken          = stderrors.New("status slug already taken")
	ErrInvalidCredentials = stderrors.New("invalid credentials")
	ErrInvalidToken       = stderrors.New("invalid token")
	ErrUnauthenticated    = stderrors.New("unauthenticated")
	ErrLimitReached       = stderrors.New("limit reached")
	ErrProviderDisabled   = stderrors.New("provider disabled")
)

type ValidationError struct {
	Field   string
	Message string
}

func (err ValidationError) Error() string {
	return err.Field + ": " + err.Message
}
