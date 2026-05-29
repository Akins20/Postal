package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// readinessProbeTimeout bounds each dependency ping during a /readyz check so a
// hung backend cannot wedge the probe.
const readinessProbeTimeout = 2 * time.Second

// mountHealth registers the liveness and readiness endpoints. /healthz reports
// process liveness (no dependencies); /readyz verifies backing dependencies.
func (s *Server) mountHealth(database, cache Pinger) {
	s.mux.Get("/healthz", handleHealthz)
	s.mux.Get("/readyz", handleReadyz(database, cache))
}

// handleHealthz reports that the process is alive. It intentionally checks no
// dependencies so orchestrators can distinguish "alive" from "ready".
func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleReadyz reports whether the server can serve traffic, pinging each
// dependency. Any failure yields 503 with per-check detail.
func handleReadyz(database, cache Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := map[string]string{}
		ready := true

		for name, dep := range map[string]Pinger{"postgres": database, "redis": cache} {
			if dep == nil {
				checks[name] = "not configured"
				ready = false
				continue
			}
			ctx, cancel := context.WithTimeout(r.Context(), readinessProbeTimeout)
			if err := dep.Ping(ctx); err != nil {
				checks[name] = "unavailable"
				ready = false
			} else {
				checks[name] = "ok"
			}
			cancel()
		}

		status := http.StatusOK
		overall := "ready"
		if !ready {
			status = http.StatusServiceUnavailable
			overall = "not ready"
		}
		writeJSON(w, status, map[string]any{"status": overall, "checks": checks})
	}
}

// writeJSON encodes v as JSON with the given status code. This is a minimal
// helper; the standard response envelope arrives in Phase 1.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
