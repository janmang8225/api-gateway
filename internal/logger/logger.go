package logger

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/janmang8225/api-gateway/internal/metrics"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

type logEntry struct {
	Time    string `json:"time"`
	Method  string `json:"method"`
	Path    string `json:"path"`
	Status  int    `json:"status"`
	Latency string `json:"latency"`
}

func Middleware(next http.Handler, m *metrics.Metrics) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := newResponseWriter(w)
		next.ServeHTTP(rw, r)

		latency := time.Since(start)

		entry := logEntry{
			Time:    time.Now().Format(time.RFC3339),
			Method:  r.Method,
			Path:    r.URL.Path,
			Status:  rw.statusCode,
			Latency: latency.String(),
		}

		out, err := json.Marshal(entry)
		if err != nil {
			log.Println("logger: failed to marshal log entry")
			return
		}

		log.Println(string(out))

		m.Record(r.URL.Path, rw.statusCode, latency)
	})
}