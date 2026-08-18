package dashboard

import (
	"encoding/json"
	"net/http"
)

// settingsSensitiveKeys are values that must never be returned to a browser.
// License credentials were intentionally removed in V2; only operational
// secrets remain protected.
var settingsSensitiveKeys = map[string]bool{
	"admin_password_hash": true,
	"ai_openai_api_key":   true,
	"threatintel_api_key": true,
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.store.ListSettings()
	if err != nil {
		http.Error(w, `{"error":"failed to get settings"}`, http.StatusInternalServerError)
		return
	}
	cleaned := make(map[string]string)
	for key, value := range settings {
		if !settingsSensitiveKeys[key] {
			cleaned[key] = value
		}
	}
	cleaned["edition"] = "free"
	cleaned["all_features_enabled"] = "true"
	writeJSON(w, http.StatusOK, cleaned)
}

// settingsProtectedKeys covers only credentials that require dedicated
// endpoints. V2 deliberately has no license-related protected values.
var settingsProtectedKeys = map[string]bool{
	"admin_password_hash": true,
	"ai_openai_api_key":   true,
	"threatintel_api_key": true,
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var settings map[string]string
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	for key, value := range settings {
		if settingsProtectedKeys[key] {
			continue
		}
		if err := s.store.SetSetting(key, value); err != nil {
			http.Error(w, `{"error":"failed to save settings"}`, http.StatusInternalServerError)
			return
		}
	}
	s.recordAudit("admin", dashboardClientIP(r), "settings_update", "success", "settings updated")
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleReloadConfig(w http.ResponseWriter, r *http.Request) {
	if s.OnConfigReload != nil {
		s.OnConfigReload()
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "reloaded",
		"message": "配置已重新加载",
	})
}
