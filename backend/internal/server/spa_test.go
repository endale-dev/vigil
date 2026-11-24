package server

import (
	"strings"
	"testing"

	"testing/fstest"
)

func TestValidateSPADoesNotRequireAssetsInDevelopment(t *testing.T) {
	if err := ValidateSPA("development", nil); err != nil {
		t.Fatalf("ValidateSPA(development, nil) = %v, want nil", err)
	}
}

func TestValidateSPARequiresEmbeddedIndexInProduction(t *testing.T) {
	for name, assets := range map[string]fstest.MapFS{
		"missing filesystem":     nil,
		"missing dist directory": {"index.html": &fstest.MapFile{Data: []byte("index")}},
		"missing index":          {"dist/assets/app.js": &fstest.MapFile{Data: []byte("app")}},
		"index is directory":     {"dist/index.html/child": &fstest.MapFile{Data: []byte("not index")}},
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidateSPA("production", assets)
			if err == nil || !strings.Contains(err.Error(), "run make web") {
				t.Fatalf("ValidateSPA(production, %v) = %v, want build instruction", assets, err)
			}
		})
	}
}

func TestValidateSPAAcceptsEmbeddedProductionIndex(t *testing.T) {
	assets := fstest.MapFS{"dist/index.html": &fstest.MapFile{Data: []byte("<!doctype html>")}}
	if err := ValidateSPA("production", assets); err != nil {
		t.Fatalf("ValidateSPA() = %v, want nil", err)
	}
}
