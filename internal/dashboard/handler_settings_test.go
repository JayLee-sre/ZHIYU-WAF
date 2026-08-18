package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"zhiyuwaf/internal/config"
	"zhiyuwaf/internal/store"
)

func newTestDashboardServer(t *testing.T) *Server {
	t.Helper()
	db, err := store.NewStore(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	cfg := config.DefaultConfig()
	cfg.Dashboard.JWTSecret = "test-secret"
	return NewServer(cfg, db)
}

func TestHealthAlwaysReportsFreeV2Features(t *testing.T) {
	server := newTestDashboardServer(t)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	server.handleHealth(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status got %d", response.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["edition"] != "free" || body["all_features_enabled"] != true {
		t.Fatalf("unexpected free feature state: %#v", body)
	}
}
