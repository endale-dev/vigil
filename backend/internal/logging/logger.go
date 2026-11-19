package logging

import (
	"io"
	"log/slog"
)

func New(environment string, output io.Writer) *slog.Logger {
	if environment == "production" {
		return slog.New(slog.NewJSONHandler(output, nil))
	}
	return slog.New(slog.NewTextHandler(output, nil))
}
