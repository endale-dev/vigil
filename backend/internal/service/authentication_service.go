package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
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

type AuthenticationResult struct {
	User         domain.User
	SessionToken string
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

func (auth *AuthenticationService) Register(ctx context.Context, name, email, password string) (AuthenticationResult, error) {
	name = strings.TrimSpace(name)
	email = normalizeEmail(email)
	if err := domain.ValidateName(name); err != nil {
		return AuthenticationResult{}, err
	}
	if err := domain.ValidateEmail(email); err != nil {
		return AuthenticationResult{}, err
	}
	if err := domain.ValidatePassword(password); err != nil {
		return AuthenticationResult{}, err
	}

	passwordHash, err := auth.dependencies.Passwords.Hash(password)
	if err != nil {
		return AuthenticationResult{}, err
	}
	for range 5 {
		statusSlug, err := newStatusSlug()
		if err != nil {
			return AuthenticationResult{}, err
		}
		user := &domain.User{
			Name:         name,
			Email:        email,
			PasswordHash: passwordHash,
			StatusSlug:   statusSlug,
			Notify:       domain.NotifySettings{EmailAlerts: true},
			CreatedAt:    auth.dependencies.Clock.Now(),
		}
		if err := auth.dependencies.Users.Create(ctx, user); err != nil {
			if errors.Is(err, apperrors.ErrSlugTaken) {
				continue
			}
			return AuthenticationResult{}, err
		}
		return auth.createSession(ctx, user)
	}
	return AuthenticationResult{}, apperrors.ErrSlugTaken
}

func (auth *AuthenticationService) Login(ctx context.Context, email, password string) (AuthenticationResult, error) {
	user, err := auth.dependencies.Users.GetByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, apperrors.ErrNotFound) {
		return AuthenticationResult{}, apperrors.ErrInvalidCredentials
	}
	if err != nil {
		return AuthenticationResult{}, err
	}
	if user.PasswordHash == "" || !auth.dependencies.Passwords.Verify(password, user.PasswordHash) {
		return AuthenticationResult{}, apperrors.ErrInvalidCredentials
	}
	return auth.createSession(ctx, user)
}

func (auth *AuthenticationService) createSession(ctx context.Context, user *domain.User) (AuthenticationResult, error) {
	rawToken, tokenHash, err := auth.dependencies.Tokens.New()
	if err != nil {
		return AuthenticationResult{}, err
	}
	now := auth.dependencies.Clock.Now()
	session := &domain.Session{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(auth.dependencies.SessionTTL),
		CreatedAt: now,
	}
	if err := auth.dependencies.Sessions.Create(ctx, session); err != nil {
		return AuthenticationResult{}, err
	}
	publicUser := *user
	publicUser.PasswordHash = ""
	return AuthenticationResult{User: publicUser, SessionToken: rawToken}, nil
}

func newStatusSlug() (string, error) {
	var random [7]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(random[:])
	return strings.ToLower(encoded[:10]), nil
}
