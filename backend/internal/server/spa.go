package server

import (
	"errors"
	"io/fs"
)

const missingSPAError = "production frontend build is missing frontend/dist/index.html; run make web"

func ValidateSPA(appEnv string, embedded fs.FS) error {
	if appEnv != "production" {
		return nil
	}
	if embedded == nil {
		return errors.New(missingSPAError)
	}
	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		return errors.New(missingSPAError)
	}
	index, err := fs.Stat(dist, "index.html")
	if err == nil && !index.IsDir() {
		return nil
	}
	return errors.New(missingSPAError)
}
