package domain

import (
	"strings"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	if got, want := NormalizeEmail("  User@Example.COM \t"), "user@example.com"; got != want {
		t.Fatalf("NormalizeEmail() = %q, want %q", got, want)
	}
}

func TestValidateEmailBoundaries(t *testing.T) {
	validMax := strings.Repeat("a", 64) + "@" + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 61)
	invalidLong := strings.Repeat("a", 64) + "@" + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 62)
	for _, email := range []string{"a@b.co", "user.name+tag@example-domain.com", validMax} {
		if err := ValidateEmail(email); err != nil {
			t.Errorf("ValidateEmail(%q) returned %v", email, err)
		}
	}
	for _, email := range []string{"", "user", "@example.com", "user@localhost", "user..name@example.com", "user@-example.com", "user@example-.com", "user name@example.com", invalidLong} {
		if err := ValidateEmail(email); err == nil {
			t.Errorf("ValidateEmail(%q) succeeded", email)
		}
	}
}

func TestValidatePasswordBoundaries(t *testing.T) {
	for _, password := range []string{strings.Repeat("a", 8), strings.Repeat("a", 128), "🔐🔐"} {
		if err := ValidatePassword(password); err != nil {
			t.Errorf("ValidatePassword(%q) returned %v", password, err)
		}
	}
	for _, password := range []string{strings.Repeat("a", 7), strings.Repeat("a", 129), "🔐"} {
		if err := ValidatePassword(password); err == nil {
			t.Errorf("ValidatePassword(%q) succeeded", password)
		}
	}
}

func TestValidateNameBoundaries(t *testing.T) {
	for _, name := range []string{"A", "  Ada Lovelace  ", strings.Repeat("界", 80)} {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) returned %v", name, err)
		}
	}
	for _, name := range []string{"", " \t ", strings.Repeat("a", 81)} {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) succeeded", name)
		}
	}
}

func TestValidateSlugBoundaries(t *testing.T) {
	for _, slug := range []string{"abc", strings.Repeat("a", 40), "a-1"} {
		if err := ValidateSlug(slug); err != nil {
			t.Errorf("ValidateSlug(%q) returned %v", slug, err)
		}
	}
	for _, slug := range []string{"", "ab", strings.Repeat("a", 41), "-abc", "abc-", "Abc", "a_b", "ébc"} {
		if err := ValidateSlug(slug); err == nil {
			t.Errorf("ValidateSlug(%q) succeeded", slug)
		}
	}
}
