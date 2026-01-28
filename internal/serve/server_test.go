package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func TestServeHandlers(t *testing.T) {
	dir := t.TempDir()
	logger := logging.NewLogger("text", httptest.NewRecorder())

	srv, err := New(dir, logger, false)
	if err != nil {
		t.Fatalf("New server: %v", err)
	}

	// Test index
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("index status: got %d", rr.Code)
	}

	// Test sessions endpoint
	req = httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	rr = httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("sessions status: got %d", rr.Code)
	}
	var sessions []interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &sessions); err != nil {
		t.Fatalf("sessions JSON: %v", err)
	}

	// Test metrics endpoint
	req = httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	rr = httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("metrics status: got %d", rr.Code)
	}
}

