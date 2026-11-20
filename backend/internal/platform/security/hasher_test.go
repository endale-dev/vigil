package security

import (
	"encoding/base64"
	"strconv"
	"strings"
	"testing"
)

func TestPasswordHasherHashAndVerify(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "correct horse battery staple"
	encoded, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != passwordHashAlgorithm || parts[1] != strconv.Itoa(passwordHashIterations) {
		t.Fatalf("Hash() = %q, want the documented PBKDF2 format", encoded)
	}
	salt, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil || len(salt) != passwordSaltSize {
		t.Fatalf("encoded salt length = %d, error = %v; want %d bytes", len(salt), err, passwordSaltSize)
	}
	key, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil || len(key) != passwordKeySize {
		t.Fatalf("encoded key length = %d, error = %v; want %d bytes", len(key), err, passwordKeySize)
	}
	if !hasher.Verify(password, encoded) {
		t.Fatal("Verify() rejected the original password")
	}
	if hasher.Verify("incorrect password", encoded) {
		t.Fatal("Verify() accepted an incorrect password")
	}
	second, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("second Hash() error = %v", err)
	}
	if encoded == second {
		t.Fatal("Hash() reused the same salt")
	}
}

func TestPasswordHasherRejectsMalformedHashes(t *testing.T) {
	hasher := NewPasswordHasher()
	malformed := []string{
		"",
		"pbkdf2-sha256",
		"pbkdf2-sha1$600000$YWJj$YWJj",
		"pbkdf2-sha256$not-a-number$YWJj$YWJj",
		"pbkdf2-sha256$1$YWJj$YWJj",
		"pbkdf2-sha256$999999999999999999999$YWJj$YWJj",
		"pbkdf2-sha256$600000$%%%$YWJj",
		"pbkdf2-sha256$600000$YWJj$%%%",
		"pbkdf2-sha256$600000$YWJj$YWJj$extra",
	}
	for _, encoded := range malformed {
		t.Run(encoded, func(t *testing.T) {
			if hasher.Verify("password", encoded) {
				t.Fatalf("Verify() accepted malformed hash %q", encoded)
			}
		})
	}
}

func TestPasswordHasherRejectsTamperedHashes(t *testing.T) {
	hasher := NewPasswordHasher()
	encoded, err := hasher.Hash("password")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	parts := strings.Split(encoded, "$")
	key, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		t.Fatalf("decode key: %v", err)
	}
	key[0] ^= 0xff
	parts[3] = base64.StdEncoding.EncodeToString(key)
	if hasher.Verify("password", strings.Join(parts, "$")) {
		t.Fatal("Verify() accepted a tampered key")
	}
}
