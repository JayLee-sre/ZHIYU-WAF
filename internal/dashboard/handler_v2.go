package dashboard

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"zhiyuwaf/internal/model"
	"zhiyuwaf/internal/store"
)

func (s *Server) v2Store(w http.ResponseWriter, r *http.Request) *store.Store {
	db, ok := s.store.(*store.Store)
	if !ok {
		writeV2Error(w, r, http.StatusServiceUnavailable, "LOCAL_STORE_REQUIRED", "V2 API requires the local SQLite store")
		return nil
	}
	return db
}

func writeV2Data(w http.ResponseWriter, r *http.Request, status int, data any) {
	writeJSON(w, status, map[string]any{
		"data":       data,
		"request_id": requestID(r),
	})
}

func writeV2Error(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
		"request_id": requestID(r),
	})
}

func requestID(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get("X-Request-ID")); value != "" && len(value) <= 128 {
		return value
	}
	return "req_" + uuid.NewString()
}

func (s *Server) handleV2DashboardSummary(w http.ResponseWriter, r *http.Request) {
	db := s.v2Store(w, r)
	if db == nil {
		return
	}
	since, _, err := v2Range(r.URL.Query().Get("range"))
	if err != nil {
		writeV2Error(w, r, http.StatusBadRequest, "INVALID_RANGE", err.Error())
		return
	}
	summary, err := db.GetV2DashboardSummary(since)
	if err != nil {
		writeV2Error(w, r, http.StatusInternalServerError, "SUMMARY_FAILED", "failed to load dashboard summary")
		return
	}
	writeV2Data(w, r, http.StatusOK, summary)
}

func (s *Server) handleV2DashboardTimeSeries(w http.ResponseWriter, r *http.Request) {
	db := s.v2Store(w, r)
	if db == nil {
		return
	}
	since, layout, err := v2Range(r.URL.Query().Get("range"))
	if err != nil {
		writeV2Error(w, r, http.StatusBadRequest, "INVALID_RANGE", err.Error())
		return
	}
	points, err := db.GetV2TimeSeries(since, layout)
	if err != nil {
		writeV2Error(w, r, http.StatusInternalServerError, "TIMESERIES_FAILED", "failed to load time series")
		return
	}
	writeV2Data(w, r, http.StatusOK, points)
}

func (s *Server) handleV2ListSites(w http.ResponseWriter, r *http.Request) {
	db := s.v2Store(w, r)
	if db == nil {
		return
	}
	sites, err := db.ListV2Sites()
	if err != nil {
		writeV2Error(w, r, http.StatusInternalServerError, "SITES_FAILED", "failed to list sites")
		return
	}
	writeV2Data(w, r, http.StatusOK, sites)
}

func (s *Server) handleV2CreateSite(w http.ResponseWriter, r *http.Request) {
	db := s.v2Store(w, r)
	if db == nil {
		return
	}
	var site model.V2Site
	if err := decodeV2JSON(w, r, &site); err != nil {
		writeV2Error(w, r, http.StatusBadRequest, "INVALID_JSON", "invalid site payload")
		return
	}
	created, err := db.CreateV2Site(site)
	if err != nil {
		writeV2Error(w, r, http.StatusBadRequest, "SITE_INVALID", err.Error())
		return
	}
	if s.OnSitesChanged != nil {
		s.OnSitesChanged()
	}
	writeV2Data(w, r, http.StatusCreated, created)
}

func (s *Server) handleV2UpdateSite(w http.ResponseWriter, r *http.Request) {
	db := s.v2Store(w, r)
	if db == nil {
		return
	}
	id, ok := v2ID(w, r, "id")
	if !ok {
		return
	}
	var site model.V2Site
	if err := decodeV2JSON(w, r, &site); err != nil {
		writeV2Error(w, r, http.StatusBadRequest, "INVALID_JSON", "invalid site payload")
		return
	}
	updated, err := db.UpdateV2Site(id, site)
	if err != nil {
		writeV2Error(w, r, http.StatusBadRequest, "SITE_INVALID", err.Error())
		return
	}
	if updated == nil {
		writeV2Error(w, r, http.StatusNotFound, "SITE_NOT_FOUND", "site not found")
		return
	}
	if s.OnSitesChanged != nil {
		s.OnSitesChanged()
	}
	writeV2Data(w, r, http.StatusOK, updated)
}

func (s *Server) handleV2DeleteSite(w http.ResponseWriter, r *http.Request) {
	db := s.v2Store(w, r)
	if db == nil {
		return
	}
	id, ok := v2ID(w, r, "id")
	if !ok {
		return
	}
	if err := db.DeleteV2Site(id); err != nil {
		writeV2Error(w, r, http.StatusInternalServerError, "SITE_DELETE_FAILED", "failed to delete site")
		return
	}
	if s.OnSitesChanged != nil {
		s.OnSitesChanged()
	}
	writeV2Data(w, r, http.StatusOK, map[string]any{"id": id, "deleted": true})
}

func (s *Server) handleV2ListEvents(w http.ResponseWriter, r *http.Request) {
	db := s.v2Store(w, r)
	if db == nil {
		return
	}
	query := r.URL.Query()
	filter := store.V2EventFilter{IP: query.Get("ip"), Category: query.Get("category"), Action: query.Get("action")}
	filter.Offset, _ = strconv.Atoi(query.Get("offset"))
	filter.Limit, _ = strconv.Atoi(query.Get("page_size"))
	if value := query.Get("site_id"); value != "" {
		filter.SiteID, _ = strconv.ParseInt(value, 10, 64)
	}
	if value := query.Get("from"); value != "" {
		filter.From, _ = time.Parse(time.RFC3339, value)
	}
	if value := query.Get("to"); value != "" {
		filter.To, _ = time.Parse(time.RFC3339, value)
	}
	events, total, err := db.ListSecurityEvents(filter)
	if err != nil {
		writeV2Error(w, r, http.StatusInternalServerError, "EVENTS_FAILED", "failed to list events")
		return
	}
	writeV2Data(w, r, http.StatusOK, map[string]any{"items": events, "total": total, "offset": filter.Offset, "page_size": filter.Limit})
}

func (s *Server) handleV2GetEvent(w http.ResponseWriter, r *http.Request) {
	db := s.v2Store(w, r)
	if db == nil {
		return
	}
	id, ok := v2ID(w, r, "id")
	if !ok {
		return
	}
	event, err := db.GetSecurityEvent(id)
	if err != nil {
		writeV2Error(w, r, http.StatusInternalServerError, "EVENT_FAILED", "failed to get event")
		return
	}
	if event == nil {
		writeV2Error(w, r, http.StatusNotFound, "EVENT_NOT_FOUND", "event not found")
		return
	}
	writeV2Data(w, r, http.StatusOK, event)
}

func (s *Server) handleV2ListBlockedIPs(w http.ResponseWriter, r *http.Request) {
	db := s.v2Store(w, r)
	if db == nil {
		return
	}
	entries, err := db.ListActiveBlockedIPs(time.Now())
	if err != nil {
		writeV2Error(w, r, http.StatusInternalServerError, "BLOCKLIST_FAILED", "failed to list blocked IPs")
		return
	}
	writeV2Data(w, r, http.StatusOK, entries)
}

func (s *Server) handleV2CreateBlockedIP(w http.ResponseWriter, r *http.Request) {
	db := s.v2Store(w, r)
	if db == nil {
		return
	}
	var payload struct {
		IP     string `json:"ip"`
		TTL    string `json:"ttl"`
		Reason string `json:"reason"`
	}
	if err := decodeV2JSON(w, r, &payload); err != nil {
		writeV2Error(w, r, http.StatusBadRequest, "INVALID_JSON", "invalid blocklist payload")
		return
	}
	ip := net.ParseIP(strings.TrimSpace(payload.IP))
	if ip == nil {
		writeV2Error(w, r, http.StatusBadRequest, "INVALID_IP", "invalid IP address")
		return
	}
	ttl := time.Duration(0)
	if strings.TrimSpace(payload.TTL) != "" {
		var err error
		ttl, err = time.ParseDuration(payload.TTL)
		if err != nil || ttl <= 0 {
			writeV2Error(w, r, http.StatusBadRequest, "INVALID_TTL", "ttl must be a positive Go duration")
			return
		}
	}
	if s.FirewallBlock == nil {
		writeV2Error(w, r, http.StatusServiceUnavailable, "FIREWALL_DEGRADED", "nftables firewall is unavailable")
		return
	}
	if err := s.FirewallBlock(ip, ttl, payload.Reason); err != nil {
		writeV2Error(w, r, http.StatusServiceUnavailable, "FIREWALL_DEGRADED", err.Error())
		return
	}
	item := model.BlockedIP{IP: ip.String(), Reason: payload.Reason, Score: 100, Source: "manual", Permanent: ttl == 0, CreatedAt: time.Now()}
	if ip.To4() != nil {
		item.Family = 4
	} else {
		item.Family = 6
	}
	if ttl > 0 {
		expires := time.Now().Add(ttl)
		item.ExpiresAt = &expires
	}
	if err := db.UpsertBlockedIP(item); err != nil {
		writeV2Error(w, r, http.StatusInternalServerError, "BLOCKLIST_SAVE_FAILED", "firewall changed but local record could not be saved")
		return
	}
	writeV2Data(w, r, http.StatusCreated, item)
}

func (s *Server) handleV2DeleteBlockedIP(w http.ResponseWriter, r *http.Request) {
	db := s.v2Store(w, r)
	if db == nil {
		return
	}
	id, ok := v2ID(w, r, "id")
	if !ok {
		return
	}
	item, err := db.DeleteBlockedIP(id)
	if err != nil {
		writeV2Error(w, r, http.StatusInternalServerError, "BLOCKLIST_DELETE_FAILED", "failed to delete blocked IP")
		return
	}
	if item == nil {
		writeV2Error(w, r, http.StatusNotFound, "BLOCKED_IP_NOT_FOUND", "blocked IP not found")
		return
	}
	if s.FirewallUnblock != nil {
		if ip := net.ParseIP(item.IP); ip != nil {
			if err := s.FirewallUnblock(ip); err != nil {
				writeV2Error(w, r, http.StatusServiceUnavailable, "FIREWALL_DEGRADED", err.Error())
				return
			}
		}
	}
	writeV2Data(w, r, http.StatusOK, map[string]any{"id": id, "deleted": true})
}

func (s *Server) handleV2SystemStatus(w http.ResponseWriter, r *http.Request) {
	status := s.FirewallStatus()
	writeV2Data(w, r, http.StatusOK, map[string]any{
		"version":           "V2",
		"uptime":            time.Since(startTime).Seconds(),
		"firewall":          status,
		"database":          "ok",
		"all_features_free": true,
	})
}

func decodeV2JSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	return json.NewDecoder(r.Body).Decode(target)
}

func v2ID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	value, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil || value <= 0 {
		writeV2Error(w, r, http.StatusBadRequest, "INVALID_ID", fmt.Sprintf("invalid %s", name))
		return 0, false
	}
	return value, true
}

func v2Range(value string) (time.Time, string, error) {
	if value == "" {
		value = "24h"
	}
	var duration time.Duration
	switch value {
	case "1h":
		duration = time.Hour
	case "6h":
		duration = 6 * time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	default:
		return time.Time{}, "", fmt.Errorf("range must be one of 1h, 6h, 24h, 7d, 30d")
	}
	layout := "hour"
	if duration >= 7*24*time.Hour {
		layout = "day"
	}
	return time.Now().Add(-duration), layout, nil
}
