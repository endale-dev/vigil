package service

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
)

type AuthenticationDependencies struct {
	Users          domain.UserRepository
	Sessions       domain.SessionRepository
	PasswordResets domain.PasswordResetRepository
	Passwords      PasswordHasher
	Tokens         TokenGenerator
	Mailer         Mailer
	Google         GoogleProvider
	Clock          Clock
	BaseURL        string
	SessionTTL     time.Duration
	Logger         *slog.Logger
}

type AuthenticationService struct {
	dependencies AuthenticationDependencies
}

func NewAuthenticationService(dependencies AuthenticationDependencies) (*AuthenticationService, error) {
	if dependencies.Users == nil || dependencies.Sessions == nil || dependencies.PasswordResets == nil ||
		dependencies.Passwords == nil || dependencies.Tokens == nil || dependencies.Mailer == nil ||
		dependencies.Clock == nil || dependencies.Logger == nil || strings.TrimSpace(dependencies.BaseURL) == "" ||
		dependencies.SessionTTL <= 0 {
		return nil, errors.New("authentication service dependencies are incomplete")
	}
	return &AuthenticationService{dependencies: dependencies}, nil
}

func normalizeEmail(email string) string {
	return domain.NormalizeEmail(email)
}
