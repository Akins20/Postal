package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestMetrics_HandlerExposesBaseCollectors(t *testing.T) {
	m := New()
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	// Go runtime collector should always be present.
	if !strings.Contains(body, "go_goroutines") {
		t.Error("expected go_goroutines in metrics output")
	}
}

func TestMetrics_MiddlewareRecordsRequests(t *testing.T) {
	m := New()
	r := chi.NewRouter()
	r.Use(m.Middleware())
	r.Get("/api/v1/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Drive one request through the route.
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil))

	// Scrape metrics and confirm the counter recorded the route pattern.
	scrape := httptest.NewRecorder()
	m.Handler().ServeHTTP(scrape, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := scrape.Body.String()

	if !strings.Contains(body, "postal_http_requests_total") {
		t.Fatal("expected postal_http_requests_total in output")
	}
	if !strings.Contains(body, `route="/api/v1/ping"`) {
		t.Errorf("expected route label /api/v1/ping in output:\n%s", body)
	}
}
