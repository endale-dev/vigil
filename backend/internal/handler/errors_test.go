package handler

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/endale-dev/vigil/internal/errors"
)

func TestWriteErrorMapsApplicationErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		status  int
		message string
	}{
		{name: "not found", err: apperrors.ErrNotFound, status: http.StatusNotFound, message: "not found"},
		{name: "provider disabled", err: apperrors.ErrProviderDisabled, status: http.StatusNotFound, message: "not found"},
		{name: "email taken", err: apperrors.ErrEmailTaken, status: http.StatusConflict, message: "email already taken"},
		{name: "slug taken", err: apperrors.ErrSlugTaken, status: http.StatusConflict, message: "status slug already taken"},
		{name: "invalid credentials", err: apperrors.ErrInvalidCredentials, status: http.StatusUnauthorized, message: "invalid credentials"},
		{name: "unauthenticated", err: apperrors.ErrUnauthenticated, status: http.StatusUnauthorized, message: "unauthenticated"},
		{name: "invalid token", err: apperrors.ErrInvalidToken, status: http.StatusBadRequest, message: "invalid token"},
		{name: "limit reached", err: apperrors.ErrLimitReached, status: http.StatusForbidden, message: "limit reached"},
		{name: "invalid JSON", err: errInvalidJSON, status: http.StatusBadRequest, message: "invalid JSON request"},
		{name: "large JSON body", err: errRequestTooLarge, status: http.StatusRequestEntityTooLarge, message: "request body is too large"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			if err := writeError(response, fmt.Errorf("service operation: %w", test.err), slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))); err != nil {
				t.Fatal(err)
			}
			if response.Code != test.status {
				t.Fatalf("got status %d, want %d", response.Code, test.status)
			}
			var body ErrorDTO
			if err := decodeJSON(response, httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(response.Body.Bytes())), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error != test.message {
				t.Fatalf("got error %q, want %q", body.Error, test.message)
			}
		})
	}
}

func TestWriteErrorMapsValidationDetails(t *testing.T) {
	response := httptest.NewRecorder()
	err := fmt.Errorf("create user: %w", apperrors.ValidationError{Field: "email", Message: "is invalid"})
	if writeErr := writeError(response, err, nil); writeErr != nil {
		t.Fatal(writeErr)
	}
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusUnprocessableEntity)
	}
	var body ErrorDTO
	if err := decodeJSON(response, httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(response.Body.Bytes())), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "is invalid" || body.Field != "email" {
		t.Fatalf("got validation response %#v", body)
	}
}

func TestWriteErrorHidesAndLogsUnexpectedDetails(t *testing.T) {
	var logs bytes.Buffer
	response := httptest.NewRecorder()
	if err := writeError(response, errors.New("MongoDB credential leaked"), slog.New(slog.NewTextHandler(&logs, nil))); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if bytes.Contains(response.Body.Bytes(), []byte("MongoDB credential leaked")) {
		t.Fatalf("response exposed internal error: %s", response.Body.String())
	}
	if !bytes.Contains(logs.Bytes(), []byte("MongoDB credential leaked")) {
		t.Fatal("internal error was not logged")
	}
}
