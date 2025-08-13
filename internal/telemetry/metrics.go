package telemetry

// --- TODOs (bite-size next steps) ---
// TODO(ryan): handlers: PUT/GET/DELETE /v1/cache/{key}?ttl=...
// TODO(ryan): validate key/ttl/value size; return appropriate status codes
// TODO(ryan): structured logging: op, key hash, duration, nodeID
// TODO(ryan): define counters/histograms for requests and latencies
// TODO(ryan): add build info gauge and uptime metric
// --- end TODOs ---

import "net/http"

// Expose a placeholder metrics handler (wire Prometheus later).
func MetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# HELP zephyrcache_placeholder 1\n# TYPE zephyrcache_placeholder counter\nzephyrcache_placeholder 1\n"))
	})
}
