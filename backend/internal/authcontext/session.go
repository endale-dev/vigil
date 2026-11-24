package authcontext

import (
	"context"

	"github.com/endale-dev/vigil/internal/service"
)

type sessionKey struct{}

func WithSession(ctx context.Context, session service.AuthenticatedSession) context.Context {
	return context.WithValue(ctx, sessionKey{}, session)
}

func SessionFrom(ctx context.Context) (service.AuthenticatedSession, bool) {
	session, ok := ctx.Value(sessionKey{}).(service.AuthenticatedSession)
	return session, ok
}
