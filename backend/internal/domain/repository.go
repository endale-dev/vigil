package domain

import (
	"context"
	"time"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id ID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByGoogleSub(ctx context.Context, sub string) (*User, error)
	GetBySlug(ctx context.Context, slug string) (*User, error)
	Update(ctx context.Context, user *User) error
}

type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	GetByTokenHash(ctx context.Context, hash string) (*Session, error)
	Delete(ctx context.Context, id ID) error
	DeleteByUserExcept(ctx context.Context, userID, keep ID) error
	DeleteByUser(ctx context.Context, userID ID) error
}

type PasswordResetRepository interface {
	Create(ctx context.Context, reset *PasswordReset) error
	GetByTokenHash(ctx context.Context, hash string) (*PasswordReset, error)
	DeleteByUser(ctx context.Context, userID ID) error
}

type MonitorClaim struct {
	Monitor    Monitor
	ClaimToken string
}

type MonitorRepository interface {
	Create(ctx context.Context, monitor *Monitor) error
	GetByID(ctx context.Context, id ID) (*Monitor, error)
	ListByOwner(ctx context.Context, owner ID) ([]Monitor, error)
	ListPublicByOwner(ctx context.Context, owner ID) ([]Monitor, error)
	CountByOwner(ctx context.Context, owner ID) (int, error)
	Update(ctx context.Context, monitor *Monitor) error
	Delete(ctx context.Context, id ID) error
	// ClaimDue reserves one due, enabled monitor. The claim expires so another worker can reclaim it after a crash.
	ClaimDue(ctx context.Context, now time.Time) (*MonitorClaim, error)
	// RecordResult must only persist the result and release the lease when claimToken matches the active claim.
	RecordResult(ctx context.Context, id ID, claimToken string, status Status, failures int, checkedAt time.Time) error
}
