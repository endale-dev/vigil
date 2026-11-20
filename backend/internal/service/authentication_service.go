package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"log/slog"
	"net/url"
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

type AuthenticatedSession struct {
	User    domain.User
	Session domain.Session
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

func (auth *AuthenticationService) Logout(ctx context.Context, sessionID domain.ID) error {
	return auth.dependencies.Sessions.Delete(ctx, sessionID)
}

func (auth *AuthenticationService) Authenticate(ctx context.Context, rawToken string) (AuthenticatedSession, error) {
	session, err := auth.dependencies.Sessions.GetByTokenHash(ctx, auth.dependencies.Tokens.Hash(rawToken))
	if err != nil {
		return AuthenticatedSession{}, apperrors.ErrUnauthenticated
	}
	if !session.ExpiresAt.After(auth.dependencies.Clock.Now()) {
		_ = auth.dependencies.Sessions.Delete(ctx, session.ID)
		return AuthenticatedSession{}, apperrors.ErrUnauthenticated
	}
	user, err := auth.dependencies.Users.GetByID(ctx, session.UserID)
	if err != nil {
		return AuthenticatedSession{}, apperrors.ErrUnauthenticated
	}
	user.PasswordHash = ""
	return AuthenticatedSession{User: *user, Session: *session}, nil
}

func (auth *AuthenticationService) ForgotPassword(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	if err := domain.ValidateEmail(email); err != nil {
		return err
	}
	user, err := auth.dependencies.Users.GetByEmail(ctx, email)
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil
	}
	if err != nil {
		auth.dependencies.Logger.ErrorContext(ctx, "forgot-password processing failed")
		return nil
	}
	if err := auth.dependencies.PasswordResets.DeleteByUser(ctx, user.ID); err != nil {
		auth.dependencies.Logger.ErrorContext(ctx, "forgot-password processing failed")
		return nil
	}
	rawToken, tokenHash, err := auth.dependencies.Tokens.New()
	if err != nil {
		auth.dependencies.Logger.ErrorContext(ctx, "forgot-password processing failed")
		return nil
	}
	now := auth.dependencies.Clock.Now()
	reset := &domain.PasswordReset{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(time.Hour),
	}
	if err := auth.dependencies.PasswordResets.Create(ctx, reset); err != nil {
		auth.dependencies.Logger.ErrorContext(ctx, "forgot-password processing failed")
		return nil
	}
	resetURL := strings.TrimRight(auth.dependencies.BaseURL, "/") + "/reset-password?token=" + url.QueryEscape(rawToken)
	if err := auth.dependencies.Mailer.Send(ctx, Mail{
		To:      user.Email,
		Subject: "Reset your Vigil password",
		Body:    "Use this link to reset your password:\n" + resetURL,
	}); err != nil {
		auth.dependencies.Logger.ErrorContext(ctx, "forgot-password email could not be sent")
	}
	return nil
}

func (auth *AuthenticationService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	reset, err := auth.dependencies.PasswordResets.GetByTokenHash(ctx, auth.dependencies.Tokens.Hash(rawToken))
	if errors.Is(err, apperrors.ErrNotFound) {
		return apperrors.ErrInvalidToken
	}
	if err != nil {
		return err
	}
	if !reset.ExpiresAt.After(auth.dependencies.Clock.Now()) {
		return apperrors.ErrInvalidToken
	}
	if err := domain.ValidatePassword(newPassword); err != nil {
		return err
	}
	passwordHash, err := auth.dependencies.Passwords.Hash(newPassword)
	if err != nil {
		return err
	}
	user, err := auth.dependencies.Users.GetByID(ctx, reset.UserID)
	if errors.Is(err, apperrors.ErrNotFound) {
		return apperrors.ErrInvalidToken
	}
	if err != nil {
		return err
	}
	user.PasswordHash = passwordHash
	if err := auth.dependencies.Users.Update(ctx, user); err != nil {
		return err
	}
	if err := auth.dependencies.PasswordResets.DeleteByUser(ctx, user.ID); err != nil {
		return err
	}
	return auth.dependencies.Sessions.DeleteByUser(ctx, user.ID)
}

func (auth *AuthenticationService) ChangePassword(ctx context.Context, userID, currentSessionID domain.ID, currentPassword, newPassword string) error {
	user, err := auth.dependencies.Users.GetByID(ctx, userID)
	if errors.Is(err, apperrors.ErrNotFound) {
		return apperrors.ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	if user.PasswordHash == "" || !auth.dependencies.Passwords.Verify(currentPassword, user.PasswordHash) {
		return apperrors.ErrInvalidCredentials
	}
	if err := domain.ValidatePassword(newPassword); err != nil {
		return err
	}
	passwordHash, err := auth.dependencies.Passwords.Hash(newPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = passwordHash
	if err := auth.dependencies.Users.Update(ctx, user); err != nil {
		return err
	}
	return auth.dependencies.Sessions.DeleteByUserExcept(ctx, user.ID, currentSessionID)
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
