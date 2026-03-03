package metrics

import (
	"net/http"
	"time"
)

// MetricsMiddleware returns HTTP middleware that records request metrics.
func MetricsMiddleware(reg *Registry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ActiveRequests.Inc()
			defer ActiveRequests.Dec()

			RequestsTotal.Inc()

			next.ServeHTTP(w, r)

			duration := time.Since(start).Seconds()
			RequestDuration.Observe(duration)
		})
	}
}
