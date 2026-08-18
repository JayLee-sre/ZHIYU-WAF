package dashboard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV2DashboardSummaryUsesEnvelope(t *testing.T) {
	server := newTestDashboardServer(t)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary?range=24h", nil)
	request.Header.Set("X-Request-ID", "test-summary")
	response := httptest.NewRecorder()
	server.handleV2DashboardSummary(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status got %d: %s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["request_id"] != "test-summary" || body["data"] == nil {
		t.Fatalf("unexpected envelope: %#v", body)
	}
}

func TestV2CreateSite(t *testing.T) {
	server := newTestDashboardServer(t)
	payload := []byte(`{"domain":"api.example.test","backend_url":"http://127.0.0.1:3000","mode":"protect","enabled":true}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/sites", bytes.NewReader(payload))
	response := httptest.NewRecorder()
	server.handleV2CreateSite(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status got %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Data struct {
			ID         int64  `json:"id"`
			Domain     string `json:"domain"`
			BackendURL string `json:"backend_url"`
			Mode       string `json:"mode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.ID <= 0 || body.Data.Domain != "api.example.test" || body.Data.BackendURL == "" || body.Data.Mode != "protect" {
		t.Fatalf("unexpected created site: %#v", body.Data)
	}
}

func TestV2SystemReportsFreeFeatures(t *testing.T) {
	server := newTestDashboardServer(t)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil)
	response := httptest.NewRecorder()
	server.handleV2SystemStatus(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status got %d", response.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok || data["all_features_free"] != true {
		t.Fatalf("unexpected system state: %#v", body)
	}
}
