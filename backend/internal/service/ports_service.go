package service

import (
	"context"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
)

type Clock interface {
	Now() time.Time
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) bool
}

type TokenGenerator interface {
	New() (raw, hash string, err error)
	Hash(raw string) string
}

type Mail struct {
	To      string
	Subject string
	Body    string
}

type Mailer interface {
	Send(ctx context.Context, mail Mail) error
}

type Alert struct {
	Monitor domain.Monitor
	Status  domain.Status
	Detail  string
	At      time.Time
}

type Notifier interface {
	Name() domain.Channel
	Enabled(user domain.User) bool
	Notify(ctx context.Context, user domain.User, alert Alert) error
}

type ProbeResult struct {
	Up      bool
	Latency time.Duration
	Detail  string
}

type Prober interface {
	Probe(ctx context.Context, monitor domain.Monitor) ProbeResult
}

type GoogleIdentity struct {
	Sub           string
	Email         string
	Name          string
	EmailVerified bool
}

type GoogleProvider interface {
	AuthURL(state, codeChallenge string) string
	Exchange(ctx context.Context, code, codeVerifier string) (GoogleIdentity, error)
}

type Alerter interface {
	Send(ctx context.Context, owner domain.User, alert Alert)
}
