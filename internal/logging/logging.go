// Package logging subsystem for metrics server and metrics agent.
package logging

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Log init base zap logger variable.
var Log *zap.Logger = zap.NewNop()

type (
	// Type for collect HTTP status code and response size.
	responseData struct {
		status int
		size   int
	}

	// Middleware type for process logging on web server response.
	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

// Middleware method for support ResponseWriter interface (Write).
func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

// Middleware method for support ResponseWriter interface (WriteHeader).
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

// Setup init zap logging.
func Setup(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	// you can select a development logger zap.NewDevelopmentConfig()
	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	zl, err := cfg.Build()
	if err != nil {
		return err
	}
	defer func() {
		_ = Log.Sync()
	}()
	Log = zl
	return nil
}

// WriteLogging main middleware function for wrap HandlerFunc and writing logs.
func WriteLogging(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}
		encCheck := r.Header.Get("Accept-Encoding")

		h(&lw, r)

		duration := time.Since(start)
		Log.Info(
			"request",
			zap.String("method", r.Method),
			zap.String("uri", r.RequestURI),
			zap.Duration("duration", duration),
			zap.Int("StatusCode", responseData.status),
			zap.Int("content-length", responseData.size),
			zap.String("Accept-Encoding", encCheck),
		)
	}
}
