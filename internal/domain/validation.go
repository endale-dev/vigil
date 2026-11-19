package domain

import (
	"strings"
	"unicode/utf8"
)

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ValidateEmail(email string) error {
	if email == "" {
		return ValidationError{Field: "email", Message: "is required"}
	}
	if len(email) > 254 {
		return ValidationError{Field: "email", Message: "must be at most 254 bytes"}
	}
	at := strings.IndexByte(email, '@')
	if at < 1 || at != strings.LastIndexByte(email, '@') || at > 64 {
		return ValidationError{Field: "email", Message: "is invalid"}
	}
	local, host := email[:at], email[at+1:]
	if local[0] == '.' || local[len(local)-1] == '.' || strings.Contains(local, "..") {
		return ValidationError{Field: "email", Message: "is invalid"}
	}
	for _, char := range local {
		if !emailLocalCharacter(char) {
			return ValidationError{Field: "email", Message: "is invalid"}
		}
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return ValidationError{Field: "email", Message: "is invalid"}
	}
	for _, label := range labels {
		if !validDomainLabel(label) {
			return ValidationError{Field: "email", Message: "is invalid"}
		}
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 128 {
		return ValidationError{Field: "password", Message: "must be between 8 and 128 bytes"}
	}
	return nil
}

func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if !utf8.ValidString(name) {
		return ValidationError{Field: "name", Message: "must contain valid Unicode text"}
	}
	length := utf8.RuneCountInString(name)
	if length < 1 || length > 80 {
		return ValidationError{Field: "name", Message: "must be between 1 and 80 characters"}
	}
	return nil
}

func ValidateSlug(slug string) error {
	if len(slug) < 3 || len(slug) > 40 || slug[0] == '-' || slug[len(slug)-1] == '-' {
		return ValidationError{Field: "slug", Message: "must be 3 to 40 lowercase letters, numbers, or hyphens without a leading or trailing hyphen"}
	}
	for _, char := range slug {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
			return ValidationError{Field: "slug", Message: "must be 3 to 40 lowercase letters, numbers, or hyphens without a leading or trailing hyphen"}
		}
	}
	return nil
}

func emailLocalCharacter(char rune) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune(".!#$%&'*+/=?^_`{|}~-", char)
}

func validDomainLabel(label string) bool {
	if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	for _, char := range label {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-') {
			return false
		}
	}
	return true
}
