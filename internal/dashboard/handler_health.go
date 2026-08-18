package dashboard

import (
	"net/http"
	"os"
	"runtime"
	"time"
)

var startTime = time.Now()

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"product":              "ZhiYu-WAF V2",
		"edition":              "free",
		"all_features_enabled": true,
		"status":               "ok",
		"hostname":             hostname,
		"uptime":               time.Since(startTime).String(),
	})
}

func (s *Server) handleHealthDetail(w http.ResponseWriter, r *http.Request) {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	hostname, _ := os.Hostname()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"product":              "ZhiYu-WAF V2",
		"edition":              "free",
		"all_features_enabled": true,
		"status":               "ok",
		"hostname":             hostname,
		"uptime":               time.Since(startTime).String(),
		"go_version":           runtime.Version(),
		"goroutines":           runtime.NumGoroutine(),
		"memory_mb":            memory.Alloc / 1024 / 1024,
		"memory_sys_mb":        memory.Sys / 1024 / 1024,
		"gc_runs":              memory.NumGC,
		"ai_enabled":           s.cfg.AI.Enabled,
		"firewall_status":      s.FirewallStatus(),
	})
}
