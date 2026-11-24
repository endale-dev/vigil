package handler

import (
	"net/http"

	"github.com/endale-dev/vigil/internal/authcontext"
	apperrors "github.com/endale-dev/vigil/internal/errors"
	"github.com/endale-dev/vigil/internal/service"
)

func authenticatedSession(request *http.Request, authentication sessionAuthenticator) (service.AuthenticatedSession, error) {
	if session, ok := authcontext.SessionFrom(request.Context()); ok {
		return session, nil
	}
	cookie, err := request.Cookie(sessionCookieName)
	if err != nil {
		return service.AuthenticatedSession{}, apperrors.ErrUnauthenticated
	}
	return authentication.Authenticate(request.Context(), cookie.Value)
}
