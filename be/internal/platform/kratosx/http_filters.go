package kratosx

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const requestIDHeader = "X-Request-Id"

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func RecoveryFilter(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Printf("[bi-api] panic method=%s path=%s err=%v", r.Method, r.URL.Path, recovered)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func AccessLogFilter(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(requestIDHeader)
			if requestID == "" {
				requestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
			}
			w.Header().Set(requestIDHeader, requestID)

			start := time.Now()
			recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)

			logger.Printf(
				"[bi-api] access request_id=%s method=%s path=%s status=%d duration=%s remote=%s",
				requestID,
				r.Method,
				r.URL.RequestURI(),
				recorder.status,
				time.Since(start).String(),
				r.RemoteAddr,
			)
		})
	}
}
