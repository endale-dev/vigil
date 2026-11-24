package server

import (
	"net/http"
	"net/http/httptest"
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

func TestSPAHandlerServesStaticFilesAndCachesAssets(t *testing.T) {
	handler := NewSPAHandler(fstest.MapFS{
		"index.html":      &fstest.MapFile{Data: []byte("<main>Vigil</main>")},
		"assets/app.js":   &fstest.MapFile{Data: []byte("console.log('vigil')")},
		"assets/site.css": &fstest.MapFile{Data: []byte("body { color: black; }")},
		"favicon.svg":     &fstest.MapFile{Data: []byte("<svg></svg>")},
	})
	for _, test := range []struct {
		path        string
		cache       string
		wantContent string
	}{
		{path: "/assets/app.js", cache: "public, max-age=31536000, immutable", wantContent: "console.log"},
		{path: "/assets/site.css", cache: "public, max-age=31536000, immutable", wantContent: "body"},
		{path: "/favicon.svg", wantContent: "<svg>"},
		{path: "/index.html", cache: "no-cache", wantContent: "Vigil"},
	} {
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != test.cache || !strings.Contains(response.Body.String(), test.wantContent) {
			t.Errorf("GET %s response/cache/body = %d/%q/%q", test.path, response.Code, response.Header().Get("Cache-Control"), response.Body.String())
		}
	}
}

func TestSPAHandlerFallsBackOnlyForFrontendRoutes(t *testing.T) {
	handler := NewSPAHandler(fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<main>Vigil</main>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('vigil')")},
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/status/team", nil))
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-cache" || !strings.Contains(response.Body.String(), "Vigil") {
		t.Fatalf("frontend route fallback = %d/%q/%q", response.Code, response.Header().Get("Cache-Control"), response.Body.String())
	}

	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/missing", nil),
		httptest.NewRequest(http.MethodGet, "/healthz", nil),
		httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil),
		httptest.NewRequest(http.MethodGet, "/missing.js", nil),
		httptest.NewRequest(http.MethodPost, "/status/team", nil),
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound || strings.Contains(response.Body.String(), "Vigil") {
			t.Errorf("%s %s fell back unexpectedly: %d/%q", request.Method, request.URL.Path, response.Code, response.Body.String())
		}
	}
}
