package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
)

func TestNewAuthenticationServiceRejectsIncompleteDependencies(t *testing.T) {
	if service, err := NewAuthenticationService(AuthenticationDependencies{}); err == nil || service != nil {
		t.Fatalf("NewAuthenticationService() = (%v, %v), want incomplete-dependency error", service, err)
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got, want := normalizeEmail("  USER@Example.Test  "), "user@example.test"; got != want {
		t.Fatalf("normalizeEmail() = %q, want %q", got, want)
	}
}

func TestAuthenticationServiceRegisterCreatesUserAndSession(t *testing.T) {
	clock := &fixedServiceClock{now: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)}
	auth, users, sessions, _, tokens := newTestAuthenticationService(t, clock)
	result, err := auth.Register(context.Background(), "  Example User  ", " User@Example.Test ", "valid-password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if result.User.Name != "Example User" || result.User.Email != "user@example.test" || result.User.Notify.EmailAlerts != true {
		t.Fatalf("registered user = %+v, want normalized fields and email alerts enabled", result.User)
	}
	if result.User.PasswordHash != "" {
		t.Fatal("Register() returned a password hash")
	}
	if result.SessionToken != "raw-session-token" || users.createCalls != 1 || tokens.newCalls != 1 {
		t.Fatalf("registration result or calls = token %q, creates %d, tokens %d", result.SessionToken, users.createCalls, tokens.newCalls)
	}
	if !regexp.MustCompile(`^[a-z2-7]{10}$`).MatchString(result.User.StatusSlug) {
		t.Fatalf("status slug %q is not 10 lowercase base32 characters", result.User.StatusSlug)
	}
	if !result.User.CreatedAt.Equal(clock.now) {
		t.Fatalf("CreatedAt = %s, want clock time %s", result.User.CreatedAt, clock.now)
	}
	if len(sessions.created) != 1 {
		t.Fatalf("created sessions = %d, want 1", len(sessions.created))
	}
	session := sessions.created[0]
	if session.UserID != result.User.ID || session.TokenHash != "stored-session-hash" ||
		!session.CreatedAt.Equal(clock.now) || !session.ExpiresAt.Equal(clock.now.Add(24*time.Hour)) {
		t.Fatalf("created session = %+v", session)
	}
	if users.stored[result.User.ID].PasswordHash != "hashed:valid-password" {
		t.Fatal("password hash was not stored on the user")
	}
}

func TestAuthenticationServiceRegisterRetriesSlugConflictsFiveTimes(t *testing.T) {
	auth, users, sessions, _, _ := newTestAuthenticationService(t, &fixedServiceClock{now: time.Now()})
	users.createErrors = []error{
		apperrors.ErrSlugTaken,
		apperrors.ErrSlugTaken,
		apperrors.ErrSlugTaken,
		apperrors.ErrSlugTaken,
		nil,
	}
	if _, err := auth.Register(context.Background(), "Example", "user@example.test", "valid-password"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if users.createCalls != 5 || len(sessions.created) != 1 {
		t.Fatalf("create calls = %d, sessions = %d; want 5 and 1", users.createCalls, len(sessions.created))
	}
}

func TestAuthenticationServiceRegisterStopsAfterFiveSlugConflicts(t *testing.T) {
	auth, users, sessions, _, _ := newTestAuthenticationService(t, &fixedServiceClock{now: time.Now()})
	users.createErrors = []error{apperrors.ErrSlugTaken, apperrors.ErrSlugTaken, apperrors.ErrSlugTaken, apperrors.ErrSlugTaken, apperrors.ErrSlugTaken, nil}
	if _, err := auth.Register(context.Background(), "Example", "user@example.test", "valid-password"); !errors.Is(err, apperrors.ErrSlugTaken) {
		t.Fatalf("Register() error = %v, want ErrSlugTaken", err)
	}
	if users.createCalls != 5 || len(sessions.created) != 0 {
		t.Fatalf("create calls = %d, sessions = %d; want 5 and 0", users.createCalls, len(sessions.created))
	}
}

func TestAuthenticationServiceRegisterValidatesBeforeWriting(t *testing.T) {
	auth, users, sessions, _, _ := newTestAuthenticationService(t, &fixedServiceClock{now: time.Now()})
	if _, err := auth.Register(context.Background(), "Example", "not-an-email", "valid-password"); err == nil {
		t.Fatal("Register() accepted an invalid email")
	}
	if users.createCalls != 0 || len(sessions.created) != 0 {
		t.Fatalf("invalid registration wrote user/session: %d/%d", users.createCalls, len(sessions.created))
	}
}

func TestAuthenticationServiceRegisterPropagatesRepositoryErrors(t *testing.T) {
	auth, users, _, _, _ := newTestAuthenticationService(t, &fixedServiceClock{now: time.Now()})
	want := errors.New("database unavailable")
	users.createErrors = []error{want}
	if _, err := auth.Register(context.Background(), "Example", "user@example.test", "valid-password"); !errors.Is(err, want) {
		t.Fatalf("Register() error = %v, want %v", err, want)
	}
}

func TestAuthenticationServiceRegisterPropagatesSessionError(t *testing.T) {
	auth, users, sessions, _, _ := newTestAuthenticationService(t, &fixedServiceClock{now: time.Now()})
	want := errors.New("session store unavailable")
	sessions.err = want
	result, err := auth.Register(context.Background(), "Example", "user@example.test", "valid-password")
	if !errors.Is(err, want) || result.SessionToken != "" {
		t.Fatalf("Register() = (%+v, %v), want session error without token", result, err)
	}
	if users.createCalls != 1 {
		t.Fatalf("user creates = %d, want 1", users.createCalls)
	}
}

func TestAuthenticationServiceLoginUsesUniformInvalidCredentials(t *testing.T) {
	auth, users, sessions, _, _ := newTestAuthenticationService(t, &fixedServiceClock{now: time.Now()})
	users.stored["password-user"] = &domain.User{
		ID: "password-user", Email: "user@example.test", PasswordHash: "hashed:valid-password",
	}
	users.stored["google-user"] = &domain.User{
		ID: "google-user", Email: "google@example.test", GoogleSub: "google-sub",
	}
	for _, test := range []struct {
		name     string
		email    string
		password string
	}{
		{name: "unknown email", email: "missing@example.test", password: "valid-password"},
		{name: "wrong password", email: " USER@example.test ", password: "wrong-password"},
		{name: "Google-only account", email: "google@example.test", password: "valid-password"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := auth.Login(context.Background(), test.email, test.password); err != apperrors.ErrInvalidCredentials {
				t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
			}
		})
	}
	if len(sessions.created) != 0 {
		t.Fatalf("failed logins created %d sessions", len(sessions.created))
	}
}

func TestAuthenticationServiceLoginCreatesSessionOnSuccess(t *testing.T) {
	clock := &fixedServiceClock{now: time.Date(2026, time.February, 3, 4, 5, 6, 0, time.UTC)}
	auth, users, sessions, _, _ := newTestAuthenticationService(t, clock)
	users.stored["password-user"] = &domain.User{
		ID: "password-user", Name: "Example", Email: "user@example.test", PasswordHash: "hashed:valid-password",
	}
	result, err := auth.Login(context.Background(), " USER@EXAMPLE.TEST ", "valid-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.User.ID != "password-user" || result.User.PasswordHash != "" || result.SessionToken != "raw-session-token" {
		t.Fatalf("Login() result = %+v", result)
	}
	if len(sessions.created) != 1 || sessions.created[0].UserID != result.User.ID || !sessions.created[0].ExpiresAt.Equal(clock.now.Add(24*time.Hour)) {
		t.Fatalf("created sessions = %+v", sessions.created)
	}
}

func newTestAuthenticationService(t *testing.T, clock Clock) (*AuthenticationService, *memoryUsers, *memorySessions, *memoryPasswordResets, *fixedTokens) {
	t.Helper()
	users := &memoryUsers{stored: make(map[domain.ID]*domain.User)}
	sessions := &memorySessions{}
	resets := &memoryPasswordResets{}
	tokens := &fixedTokens{}
	service, err := NewAuthenticationService(AuthenticationDependencies{
		Users: users, Sessions: sessions, PasswordResets: resets,
		Passwords: fixedPasswords{}, Tokens: tokens, Mailer: fixedMailer{},
		Clock: clock, BaseURL: "https://vigil.example.test", SessionTTL: 24 * time.Hour,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("NewAuthenticationService() error = %v", err)
	}
	return service, users, sessions, resets, tokens
}

type fixedServiceClock struct{ now time.Time }

func (clock *fixedServiceClock) Now() time.Time { return clock.now }

type fixedPasswords struct{}

func (fixedPasswords) Hash(password string) (string, error) { return "hashed:" + password, nil }
func (fixedPasswords) Verify(password, hash string) bool    { return hash == "hashed:"+password }

type fixedTokens struct{ newCalls int }

func (tokens *fixedTokens) New() (string, string, error) {
	tokens.newCalls++
	return "raw-session-token", "stored-session-hash", nil
}
func (*fixedTokens) Hash(raw string) string { return "hashed:" + raw }

type fixedMailer struct{}

func (fixedMailer) Send(context.Context, Mail) error { return nil }

type memoryUsers struct {
	stored       map[domain.ID]*domain.User
	createErrors []error
	createCalls  int
}

func (repository *memoryUsers) Create(_ context.Context, user *domain.User) error {
	repository.createCalls++
	index := repository.createCalls - 1
	if index < len(repository.createErrors) && repository.createErrors[index] != nil {
		return repository.createErrors[index]
	}
	if user.ID == "" {
		user.ID = domain.ID(fmt.Sprintf("user-%d", repository.createCalls))
	}
	copyOfUser := *user
	repository.stored[user.ID] = &copyOfUser
	return nil
}

func (repository *memoryUsers) GetByID(_ context.Context, id domain.ID) (*domain.User, error) {
	if user, ok := repository.stored[id]; ok {
		copyOfUser := *user
		return &copyOfUser, nil
	}
	return nil, apperrors.ErrNotFound
}

func (repository *memoryUsers) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, user := range repository.stored {
		if user.Email == email {
			copyOfUser := *user
			return &copyOfUser, nil
		}
	}
	return nil, apperrors.ErrNotFound
}

func (*memoryUsers) GetByGoogleSub(context.Context, string) (*domain.User, error) {
	return nil, apperrors.ErrNotFound
}

func (*memoryUsers) GetBySlug(context.Context, string) (*domain.User, error) {
	return nil, apperrors.ErrNotFound
}

func (repository *memoryUsers) Update(_ context.Context, user *domain.User) error {
	copyOfUser := *user
	repository.stored[user.ID] = &copyOfUser
	return nil
}

type memorySessions struct {
	created []*domain.Session
	err     error
}

func (repository *memorySessions) Create(_ context.Context, session *domain.Session) error {
	if repository.err != nil {
		return repository.err
	}
	copyOfSession := *session
	repository.created = append(repository.created, &copyOfSession)
	return nil
}

func (*memorySessions) GetByTokenHash(context.Context, string) (*domain.Session, error) {
	return nil, apperrors.ErrNotFound
}
func (*memorySessions) Delete(context.Context, domain.ID) error { return nil }
func (*memorySessions) DeleteByUserExcept(context.Context, domain.ID, domain.ID) error {
	return nil
}
func (*memorySessions) DeleteByUser(context.Context, domain.ID) error { return nil }

type memoryPasswordResets struct{}

func (*memoryPasswordResets) Create(context.Context, *domain.PasswordReset) error { return nil }
func (*memoryPasswordResets) GetByTokenHash(context.Context, string) (*domain.PasswordReset, error) {
	return nil, apperrors.ErrNotFound
}
func (*memoryPasswordResets) DeleteByUser(context.Context, domain.ID) error { return nil }

var _ domain.UserRepository = (*memoryUsers)(nil)
var _ domain.SessionRepository = (*memorySessions)(nil)
var _ domain.PasswordResetRepository = (*memoryPasswordResets)(nil)
