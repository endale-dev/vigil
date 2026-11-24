package server

import (
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

const missingSPAError = "production frontend build is missing frontend/dist/index.html; run make web"

func ValidateSPA(appEnv string, embedded fs.FS) error {
	_, err := embeddedSPA(appEnv, embedded)
	return err
}

func embeddedSPA(appEnv string, embedded fs.FS) (fs.FS, error) {
	if appEnv != "production" {
		return nil, nil
	}
	if embedded == nil {
		return nil, errors.New(missingSPAError)
	}
	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		return nil, errors.New(missingSPAError)
	}
	index, err := fs.Stat(dist, "index.html")
	if err == nil && !index.IsDir() {
		return dist, nil
	}
	return nil, errors.New(missingSPAError)
}

func NewSPAHandler(assets fs.FS) http.Handler {
	if assets == nil {
		return http.NotFoundHandler()
	}
	return &spaHandler{assets: assets, files: http.FS(assets)}
}

type spaHandler struct {
	assets fs.FS
	files  http.FileSystem
}

func (handler *spaHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	requestPath := path.Clean("/" + strings.TrimPrefix(request.URL.Path, "/"))
	if requestPath == "/api" || strings.HasPrefix(requestPath, "/api/") || requestPath == "/healthz" {
		http.NotFound(writer, request)
		return
	}
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		http.NotFound(writer, request)
		return
	}

	assetPath := strings.TrimPrefix(requestPath, "/")
	if assetPath == "" {
		assetPath = "index.html"
	}
	if fs.ValidPath(assetPath) {
		if info, err := fs.Stat(handler.assets, assetPath); err == nil && !info.IsDir() {
			handler.serveFile(writer, request, assetPath)
			return
		}
	}
	if requestPath == "/assets" || strings.HasPrefix(requestPath, "/assets/") || path.Ext(requestPath) != "" {
		http.NotFound(writer, request)
		return
	}
	handler.serveFile(writer, request, "index.html")
}

func (handler *spaHandler) serveFile(writer http.ResponseWriter, request *http.Request, filename string) {
	file, err := handler.files.Open(filename)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(writer, request)
		return
	}
	if filename == "index.html" {
		writer.Header().Set("Cache-Control", "no-cache")
	} else if strings.HasPrefix("/"+filename, "/assets/") {
		writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	http.ServeContent(writer, request, filename, info.ModTime(), file)
}
