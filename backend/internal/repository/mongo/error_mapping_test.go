package mongo

import (
	"errors"
	"testing"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
)

func TestIDConversion(t *testing.T) {
	const hexID = "507f1f77bcf86cd799439011"
	objectID, err := idToObjectID(domain.ID(hexID))
	if err != nil {
		t.Fatal(err)
	}
	if got := objectIDToID(objectID); got != domain.ID(hexID) {
		t.Fatalf("objectIDToID() = %q, want %q", got, hexID)
	}
	if _, err := idToObjectID("invalid"); !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("idToObjectID() error = %v, want ErrNotFound", err)
	}
}

func TestMapError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "nil", err: nil},
		{name: "no documents", err: driver.ErrNoDocuments, want: apperrors.ErrNotFound},
		{name: "email duplicate", err: driver.WriteException{WriteErrors: driver.WriteErrors{{Code: 11000, Message: "duplicate key index: users_email_unique"}}}, want: apperrors.ErrEmailTaken},
		{name: "slug duplicate", err: driver.WriteException{WriteErrors: driver.WriteErrors{{Code: 11000, Message: "duplicate key index: users_status_slug_unique"}}}, want: apperrors.ErrSlugTaken},
		{name: "unrelated duplicate", err: driver.WriteException{WriteErrors: driver.WriteErrors{{Code: 11000, Message: "duplicate key index: monitors_owner"}}}},
		{name: "unknown error", err: errors.New("private driver details")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapError(tt.err)
			if tt.want == nil {
				if tt.err == nil && got != nil {
					t.Fatalf("mapError(nil) = %v", got)
				}
				if tt.err != nil {
					if got == nil || errors.Is(got, apperrors.ErrEmailTaken) || errors.Is(got, apperrors.ErrSlugTaken) {
						t.Fatalf("unmapped error = %v", got)
					}
					if got.Error() == tt.err.Error() {
						t.Fatalf("driver error details were exposed: %v", got)
					}
				}
				return
			}
			if !errors.Is(got, tt.want) {
				t.Fatalf("mapError() = %v, want %v", got, tt.want)
			}
		})
	}
}
