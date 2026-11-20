package security

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
)

func TestTokenGeneratorNew(t *testing.T) {
	generator := NewTokenGenerator()
	raw, hash, err := generator.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if strings.Contains(raw, "=") {
		t.Fatalf("token %q contains base64 padding", raw)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode token: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("decoded token length = %d, want 32", len(decoded))
	}
	if len(hash) != sha256.Size*2 {
		t.Fatalf("hash length = %d, want %d hex characters", len(hash), sha256.Size*2)
	}
	if _, err := hex.DecodeString(hash); err != nil {
		t.Fatalf("hash %q is not hexadecimal: %v", hash, err)
	}
	if got := generator.Hash(raw); got != hash {
		t.Fatalf("Hash(raw) = %q, New() hash = %q", got, hash)
	}
}

func TestTokenGeneratorReturnsUniqueTokens(t *testing.T) {
	generator := NewTokenGenerator()
	first, firstHash, err := generator.New()
	if err != nil {
		t.Fatalf("first New() error = %v", err)
	}
	second, secondHash, err := generator.New()
	if err != nil {
		t.Fatalf("second New() error = %v", err)
	}
	if first == second || firstHash == secondHash {
		t.Fatal("New() reused a token or its hash")
	}
}

func TestTokenGeneratorHashIsDeterministic(t *testing.T) {
	generator := NewTokenGenerator()
	raw := "opaque-token"
	wantDigest := sha256.Sum256([]byte(raw))
	want := hex.EncodeToString(wantDigest[:])
	if got := generator.Hash(raw); got != want {
		t.Fatalf("Hash() = %q, want %q", got, want)
	}
	if got := generator.Hash(raw); got != want {
		t.Fatalf("second Hash() = %q, want %q", got, want)
	}
}
