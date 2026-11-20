package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"github.com/endale-dev/vigil/internal/service"
)

type TokenGenerator struct{}

func NewTokenGenerator() *TokenGenerator {
	return &TokenGenerator{}
}

func (generator *TokenGenerator) New() (raw, hash string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(bytes)
	return raw, generator.Hash(raw), nil
}

func (*TokenGenerator) Hash(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(digest[:])
}

var _ service.TokenGenerator = (*TokenGenerator)(nil)
