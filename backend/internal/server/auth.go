package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/endale-dev/vigil/internal/authcontext"
	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
	"github.com/endale-dev/vigil/internal/service"
)

const sessionCookieName = "vigil_session"

type sessionAuthenticator interface {
	Authenticate(context.Context, string) (service.AuthenticatedSession, error)
}

func UserFrom(ctx context.Context) (domain.User, bool) {
	session, ok := authcontext.SessionFrom(ctx)
	return session.User, ok
}

func RequireAuth(authentication sessionAuthenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if authentication == nil {
				writeServerError(writer, http.StatusInternalServerError, "internal server error")
				return
			}
			cookie, err := request.Cookie(sessionCookieName)
			if err != nil {
				writeServerError(writer, http.StatusUnauthorized, "unauthenticated")
				return
			}
			session, err := authentication.Authenticate(request.Context(), cookie.Value)
			if err != nil {
				if errors.Is(err, apperrors.ErrUnauthenticated) {
					writeServerError(writer, http.StatusUnauthorized, "unauthenticated")
				} else {
					writeServerError(writer, http.StatusInternalServerError, "internal server error")
				}
				return
			}
			ctx := authcontext.WithSession(request.Context(), session)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

func writeServerError(writer http.ResponseWriter, status int, message string) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(struct {
		Error string `json:"error"`
	}{Error: message})
}
