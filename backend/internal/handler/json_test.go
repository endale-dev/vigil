package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr error
	}{
		{name: "valid object", body: `{"name":"Ada"}`},
		{name: "unknown field", body: `{"name":"Ada","extra":true}`, wantErr: errInvalidJSON},
		{name: "trailing object", body: `{"name":"Ada"} {"name":"Grace"}`, wantErr: errInvalidJSON},
		{name: "malformed object", body: `{"name":`, wantErr: errInvalidJSON},
		{name: "empty body", body: "", wantErr: errInvalidJSON},
		{name: "body over limit", body: `{"name":"Ada"}` + strings.Repeat(" ", maxJSONBodySize+1), wantErr: errRequestTooLarge},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			var destination struct {
				Name string `json:"name"`
			}
			err := decodeJSON(httptest.NewRecorder(), request, &destination)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("got error %v, want %v", err, test.wantErr)
			}
			if test.wantErr == nil && destination.Name != "Ada" {
				t.Fatalf("got name %q, want Ada", destination.Name)
			}
		})
	}
}

func TestDecodeJSONAcceptsBodyAtLimit(t *testing.T) {
	body := `{"name":"Ada"}`
	body += strings.Repeat(" ", maxJSONBodySize-len(body))
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	var destination struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(httptest.NewRecorder(), request, &destination); err != nil {
		t.Fatal(err)
	}
}

func TestWriteJSON(t *testing.T) {
	response := httptest.NewRecorder()
	if err := writeJSON(response, http.StatusCreated, struct {
		Name string `json:"name"`
	}{Name: "Ada"}); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusCreated {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusCreated)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("got content type %q", got)
	}
	if got, want := response.Body.String(), "{\"name\":\"Ada\"}\n"; got != want {
		t.Fatalf("got body %q, want %q", got, want)
	}
}

func TestWriteJSONMarshalsBeforeWriting(t *testing.T) {
	response := httptest.NewRecorder()
	err := writeJSON(response, http.StatusOK, make(chan int))
	if err == nil {
		t.Fatal("expected unsupported value error")
	}
	if response.Body.Len() != 0 || response.Header().Get("Content-Type") != "" {
		t.Fatalf("response was written before encoding succeeded: %#v", response)
	}
}
