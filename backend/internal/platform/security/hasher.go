package security

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/endale-dev/vigil/internal/service"
)

const (
	passwordHashAlgorithm  = "pbkdf2-sha256"
	passwordHashIterations = 600_000
	passwordSaltSize       = 16
	passwordKeySize        = 32
)

type PasswordHasher struct{}

func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{}
}

func (*PasswordHasher) Hash(password string) (string, error) {
	salt := make([]byte, passwordSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordHashIterations, passwordKeySize)
	if err != nil {
		return "", err
	}
	return strings.Join([]string{
		passwordHashAlgorithm,
		strconv.Itoa(passwordHashIterations),
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(key),
	}, "$"), nil
}

func (*PasswordHasher) Verify(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != passwordHashAlgorithm {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations != passwordHashIterations {
		return false
	}
	salt, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil || len(salt) != passwordSaltSize {
		return false
	}
	want, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil || len(want) != passwordKeySize {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iterations, passwordKeySize)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}

var _ service.PasswordHasher = (*PasswordHasher)(nil)
