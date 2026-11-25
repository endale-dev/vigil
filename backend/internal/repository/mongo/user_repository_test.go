package mongo

import (
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
)

func TestUserDocumentMapping(t *testing.T) {
	want := &domain.User{
		ID:           "507f1f77bcf86cd799439011",
		Name:         "Ada",
		Email:        "ada@example.com",
		PasswordHash: "hash",
		GoogleSub:    "google-sub",
		StatusSlug:   "ada-status",
		Notify: domain.NotifySettings{
			EmailAlerts:      true,
			EmailDestination: "alerts@example.com",
			TelegramChatID:   "12345",
			TelegramDisabled: true,
		},
		CreatedAt: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
	}
	document, err := userToDocument(want)
	if err != nil {
		t.Fatal(err)
	}
	got := document.toDomain()
	if *got != *want {
		t.Fatalf("user mapping mismatch: got %+v, want %+v", got, want)
	}
}
