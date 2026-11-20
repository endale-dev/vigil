package mongo

import (
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
)

func TestPasswordResetDocumentMapping(t *testing.T) {
	want := &domain.PasswordReset{
		ID:        "507f1f77bcf86cd799439011",
		UserID:    "507f1f77bcf86cd799439012",
		TokenHash: "token-hash",
		ExpiresAt: time.Date(2026, time.January, 3, 4, 5, 6, 0, time.UTC),
	}
	document, err := passwordResetToDocument(want)
	if err != nil {
		t.Fatal(err)
	}
	got := document.toDomain()
	if *got != *want {
		t.Fatalf("password reset mapping mismatch: got %+v, want %+v", got, want)
	}
}
