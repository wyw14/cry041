package httptransport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublishRequiresReleaseManager(t *testing.T) {
	router := NewRouter(nil, nil, func() error { return nil })
	req := httptest.NewRequest(http.MethodPost, "/api/v1/releases/r1/publish", nil)
	req.Header.Set("If-Match", "1")
	req.Header.Set("X-Actor-Role", "viewer")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "FORBIDDEN") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestCreateRejectsMissingIdempotencyAndFields(t *testing.T) {
	router := NewRouter(nil, nil, func() error { return nil })
	req := httptest.NewRequest(http.MethodPost, "/api/v1/releases", strings.NewReader(`{"risk":"high"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "VALIDATION_FAILED") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
