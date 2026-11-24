package server

import (
	"log/slog"
	"net/http"
	"time"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			defer func() {
				if recover() != nil {
					if logger != nil {
						logger.ErrorContext(request.Context(), "HTTP handler panic")
					}
					writeServerError(writer, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(writer, request)
		})
	}
}

func RequestLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			started := time.Now()
			response := &loggedResponseWriter{ResponseWriter: writer}
			defer func() {
				if logger != nil {
					status := response.status
					if status == 0 {
						status = http.StatusOK
					}
					logger.InfoContext(request.Context(), "HTTP request",
						"method", request.Method,
						"path", request.URL.Path,
						"status", status,
						"duration", time.Since(started),
					)
				}
			}()
			next.ServeHTTP(response, request)
		})
	}
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(writer, request)
	})
}

func CSRF(next http.Handler) http.Handler {
	return http.NewCrossOriginProtection().Handler(next)
}

type loggedResponseWriter struct {
	http.ResponseWriter
	status int
}

func (writer *loggedResponseWriter) WriteHeader(status int) {
	if writer.status != 0 {
		return
	}
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *loggedResponseWriter) Write(body []byte) (int, error) {
	if writer.status == 0 {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.ResponseWriter.Write(body)
}
