package logger

import (
	"lb4a/types"
	"log/slog"
	"net/http"
	"time"
)

type ResponseRecorder struct {
	http.ResponseWriter
	StatusCode int
}

func (Rec *ResponseRecorder) WriteHeader(code int) {
	Rec.StatusCode = code
	Rec.ResponseWriter.WriteHeader(code)
}

// 3. The Middleware function
func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the writer to capture the status code (default to 200)
		recorder := &ResponseRecorder{
			ResponseWriter: w,
			StatusCode:     http.StatusOK,
		}

		// Let your MannualProxy do its work
		next(recorder, r)

		// Calculate latency
		duration := time.Since(start)

		// Log the structured JSON data
		types.Log.Info("Access Log",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.StatusCode),
			slog.String("duration", duration.String()),
			slog.String("client_ip", r.RemoteAddr),
		)
	}
}
