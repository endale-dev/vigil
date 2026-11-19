package domain

import "testing"

func TestValidationError(t *testing.T) {
	err := ValidationError{Field: "email", Message: "is required"}
	if got, want := err.Error(), "email: is required"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}
