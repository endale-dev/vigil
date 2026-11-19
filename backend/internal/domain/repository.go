package domain

import "context"

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
