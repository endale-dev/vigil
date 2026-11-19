package domain

import "time"

type Session struct {
	ID        ID
	UserID    ID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type PasswordReset struct {
	ID        ID
	UserID    ID
	TokenHash string
	ExpiresAt time.Time
}
