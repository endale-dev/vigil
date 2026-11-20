package service

import "testing"

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
