package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"

	"zhiyuwaf/internal/model"
)

// V2EventFilter provides indexed, bounded event queries for the V2 API.
type V2EventFilter struct {
	SiteID   int64
	IP       string
	Category string
	Action   string
	From     time.Time
	To       time.Time
	Offset   int
	Limit    int
}

func (s *Store) InsertSecurityEvent(event model.SecurityEvent) error {
	if event.ClientIP == "" {
		return fmt.Errorf("client IP is required")
	}
	if event.Action == "" {
		event.Action = "log"
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	result, err := s.db.Exec(`INSERT INTO events(request_id, site_id, client_ip, method, host, path, rule_id, category, severity, confidence, risk_score, action, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.RequestID, event.SiteID, event.ClientIP, event.Method, event.Host, event.Path, event.RuleID, event.Category, event.Severity, event.Confidence, event.RiskScore, event.Action, redactUserAgent(event.UserAgent), event.CreatedAt)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	event.ID = id
	return nil
}

func (s *Store) ListSecurityEvents(filter V2EventFilter) ([]model.SecurityEvent, int, error) {
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	where, args := v2EventWhere(filter)
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM events"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Query(`SELECT id, request_id, site_id, client_ip, method, host, path, rule_id, category, severity, confidence, risk_score, action, user_agent, created_at FROM events`+where+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`, append(args, filter.Limit, filter.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]model.SecurityEvent, 0)
	for rows.Next() {
		var item model.SecurityEvent
		if err := rows.Scan(&item.ID, &item.RequestID, &item.SiteID, &item.ClientIP, &item.Method, &item.Host, &item.Path, &item.RuleID, &item.Category, &item.Severity, &item.Confidence, &item.RiskScore, &item.Action, &item.UserAgent, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (s *Store) GetSecurityEvent(id int64) (*model.SecurityEvent, error) {
	if id <= 0 {
		return nil, fmt.Errorf("event ID must be positive")
	}
	var item model.SecurityEvent
	err := s.db.QueryRow(`SELECT id, request_id, site_id, client_ip, method, host, path, rule_id, category, severity, confidence, risk_score, action, user_agent, created_at FROM events WHERE id = ?`, id).Scan(
		&item.ID, &item.RequestID, &item.SiteID, &item.ClientIP, &item.Method, &item.Host, &item.Path, &item.RuleID, &item.Category, &item.Severity, &item.Confidence, &item.RiskScore, &item.Action, &item.UserAgent, &item.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// UpsertBlockedIP creates or refreshes a local blocklist entry. IP parsing is
// mandatory so callers can never persist an arbitrary firewall expression.
func (s *Store) UpsertBlockedIP(item model.BlockedIP) error {
	parsed := net.ParseIP(strings.TrimSpace(item.IP))
	if parsed == nil {
		return fmt.Errorf("invalid IP address")
	}
	if item.Family == 0 {
		if parsed.To4() != nil {
			item.Family = 4
		} else {
			item.Family = 6
		}
	}
	if item.Family != 4 && item.Family != 6 {
		return fmt.Errorf("IP family must be 4 or 6")
	}
	if item.Source == "" {
		item.Source = "local"
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	item.IP = parsed.String()
	_, err := s.db.Exec(`INSERT INTO blocked_ips(ip, family, reason, score, expires_at, permanent, source, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(ip) DO UPDATE SET family=excluded.family, reason=excluded.reason, score=excluded.score, expires_at=excluded.expires_at, permanent=excluded.permanent, source=excluded.source`,
		item.IP, item.Family, item.Reason, item.Score, item.ExpiresAt, item.Permanent, item.Source, item.CreatedAt)
	return err
}

func (s *Store) ListActiveBlockedIPs(now time.Time) ([]model.BlockedIP, error) {
	rows, err := s.db.Query(`SELECT id, ip, family, reason, score, expires_at, permanent, source, created_at FROM blocked_ips WHERE permanent = 1 OR expires_at IS NULL OR expires_at > ? ORDER BY created_at DESC`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.BlockedIP, 0)
	for rows.Next() {
		var item model.BlockedIP
		var expires sql.NullTime
		if err := rows.Scan(&item.ID, &item.IP, &item.Family, &item.Reason, &item.Score, &expires, &item.Permanent, &item.Source, &item.CreatedAt); err != nil {
			return nil, err
		}
		if expires.Valid {
			value := expires.Time
			item.ExpiresAt = &value
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) DeleteBlockedIP(id int64) (*model.BlockedIP, error) {
	if id <= 0 {
		return nil, fmt.Errorf("blocked IP ID must be positive")
	}
	var item model.BlockedIP
	var expires sql.NullTime
	err := s.db.QueryRow(`SELECT id, ip, family, reason, score, expires_at, permanent, source, created_at FROM blocked_ips WHERE id = ?`, id).Scan(&item.ID, &item.IP, &item.Family, &item.Reason, &item.Score, &expires, &item.Permanent, &item.Source, &item.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if expires.Valid {
		value := expires.Time
		item.ExpiresAt = &value
	}
	if _, err := s.db.Exec("DELETE FROM blocked_ips WHERE id = ?", id); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) RecordRiskEvent(item model.RiskEvent) error {
	if net.ParseIP(strings.TrimSpace(item.ClientIP)) == nil {
		return fmt.Errorf("invalid client IP")
	}
	if item.Level == "" || item.Action == "" {
		return fmt.Errorf("risk level and action are required")
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	_, err := s.db.Exec(`INSERT INTO risk_events(client_ip, score, level, reason, action, created_at) VALUES (?, ?, ?, ?, ?, ?)`, item.ClientIP, item.Score, item.Level, item.Reason, item.Action, item.CreatedAt)
	return err
}

// ListV2Sites exposes the V2 view of the existing local sites table. rowid is
// used as its stable SQLite integer identity while legacy string IDs remain
// intact for a rolling migration.
func (s *Store) ListV2Sites() ([]model.V2Site, error) {
	rows, err := s.db.Query(`SELECT rowid, domain, backend_url, mode, enabled, created_at, updated_at FROM sites WHERE domain <> '' ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.V2Site, 0)
	for rows.Next() {
		var site model.V2Site
		if err := rows.Scan(&site.ID, &site.Domain, &site.BackendURL, &site.Mode, &site.Enabled, &site.CreatedAt, &site.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, site)
	}
	return out, rows.Err()
}

func (s *Store) CleanupV2Data(retention time.Duration, now time.Time) (int64, error) {
	if retention <= 0 {
		return 0, nil
	}
	result, err := s.db.Exec("DELETE FROM events WHERE created_at < ?", now.Add(-retention))
	if err != nil {
		return 0, err
	}
	if _, err := s.db.Exec("DELETE FROM risk_events WHERE created_at < ?", now.Add(-retention)); err != nil {
		return 0, err
	}
	if _, err := s.db.Exec("DELETE FROM blocked_ips WHERE permanent = 0 AND expires_at IS NOT NULL AND expires_at <= ?", now); err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func v2EventWhere(filter V2EventFilter) (string, []any) {
	clauses := make([]string, 0, 6)
	args := make([]any, 0, 6)
	if filter.SiteID > 0 {
		clauses, args = append(clauses, "site_id = ?"), append(args, filter.SiteID)
	}
	if filter.IP != "" {
		clauses, args = append(clauses, "client_ip = ?"), append(args, filter.IP)
	}
	if filter.Category != "" {
		clauses, args = append(clauses, "category = ?"), append(args, filter.Category)
	}
	if filter.Action != "" {
		clauses, args = append(clauses, "action = ?"), append(args, filter.Action)
	}
	if !filter.From.IsZero() {
		clauses, args = append(clauses, "created_at >= ?"), append(args, filter.From)
	}
	if !filter.To.IsZero() {
		clauses, args = append(clauses, "created_at <= ?"), append(args, filter.To)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func redactUserAgent(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 512 {
		return value[:512]
	}
	return value
}

// V2DashboardSummary is the compact local-first overview used by the V2 UI.
type V2DashboardSummary struct {
	RequestCount  int                   `json:"request_count"`
	BlockedCount  int                   `json:"blocked_count"`
	AttackIPCount int                   `json:"attack_ip_count"`
	HighRiskCount int                   `json:"high_risk_count"`
	TopCategories []V2CategoryCount     `json:"top_categories"`
	RecentEvents  []model.SecurityEvent `json:"recent_events"`
}

type V2CategoryCount struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

type V2TimePoint struct {
	Timestamp string `json:"timestamp"`
	Requests  int    `json:"requests"`
	Blocked   int    `json:"blocked"`
	HighRisk  int    `json:"high_risk"`
}

func (s *Store) GetV2DashboardSummary(since time.Time) (*V2DashboardSummary, error) {
	summary := &V2DashboardSummary{TopCategories: make([]V2CategoryCount, 0), RecentEvents: make([]model.SecurityEvent, 0)}
	if err := s.db.QueryRow(`SELECT COUNT(*),
		COALESCE(SUM(CASE WHEN action = 'block' THEN 1 ELSE 0 END), 0),
		COUNT(DISTINCT CASE WHEN risk_score >= 30 THEN client_ip END),
		COALESCE(SUM(CASE WHEN risk_score >= 60 THEN 1 ELSE 0 END), 0)
		FROM events WHERE created_at >= ?`, since).Scan(&summary.RequestCount, &summary.BlockedCount, &summary.AttackIPCount, &summary.HighRiskCount); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT category, COUNT(*) FROM events WHERE created_at >= ? AND category <> '' GROUP BY category ORDER BY COUNT(*) DESC, category ASC LIMIT 5`, since)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item V2CategoryCount
		if err := rows.Scan(&item.Category, &item.Count); err != nil {
			rows.Close()
			return nil, err
		}
		summary.TopCategories = append(summary.TopCategories, item)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	recent, _, err := s.ListSecurityEvents(V2EventFilter{From: since, Limit: 8})
	if err != nil {
		return nil, err
	}
	summary.RecentEvents = recent
	return summary, nil
}

// GetV2TimeSeries returns hour/day buckets selected by bucketLayout. The SQL
// expression is fixed internally and never derives from user input.
func (s *Store) GetV2TimeSeries(since time.Time, bucketLayout string) ([]V2TimePoint, error) {
	format := "%Y-%m-%d %H:00:00"
	if bucketLayout == "day" {
		format = "%Y-%m-%d 00:00:00"
	}
	rows, err := s.db.Query(`SELECT strftime(?, created_at) AS bucket, COUNT(*),
		COALESCE(SUM(CASE WHEN action = 'block' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN risk_score >= 60 THEN 1 ELSE 0 END), 0)
		FROM events WHERE created_at >= ? GROUP BY bucket ORDER BY bucket ASC`, format, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]V2TimePoint, 0)
	for rows.Next() {
		var point V2TimePoint
		if err := rows.Scan(&point.Timestamp, &point.Requests, &point.Blocked, &point.HighRisk); err != nil {
			return nil, err
		}
		out = append(out, point)
	}
	return out, rows.Err()
}

func (s *Store) CreateV2Site(site model.V2Site) (*model.V2Site, error) {
	site.Domain = normalizeV2Domain(site.Domain)
	if site.Domain == "" || strings.TrimSpace(site.BackendURL) == "" {
		return nil, fmt.Errorf("domain and backend_url are required")
	}
	if site.Mode == "" {
		site.Mode = "protect"
	}
	if !validV2Mode(site.Mode) {
		return nil, fmt.Errorf("invalid site mode")
	}
	now := time.Now()
	domains, _ := json.Marshal([]string{site.Domain})
	result, err := s.db.Exec(`INSERT INTO sites(id, name, domains, upstream, enabled, ai_enabled, challenge_enabled, maintenance_mode, site_type, domain, backend_url, mode, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, 0, ?, 'website', ?, ?, ?, ?, ?)`,
		uuid.NewString(), site.Domain, string(domains), site.BackendURL, site.Enabled, site.Mode == "emergency", site.Domain, site.BackendURL, site.Mode, now, now)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	site.ID, site.CreatedAt, site.UpdatedAt = id, now, now
	return &site, nil
}

func (s *Store) UpdateV2Site(id int64, site model.V2Site) (*model.V2Site, error) {
	if id <= 0 {
		return nil, fmt.Errorf("site ID must be positive")
	}
	site.Domain = normalizeV2Domain(site.Domain)
	if site.Domain == "" || strings.TrimSpace(site.BackendURL) == "" || !validV2Mode(site.Mode) {
		return nil, fmt.Errorf("valid domain, backend_url and mode are required")
	}
	domains, _ := json.Marshal([]string{site.Domain})
	now := time.Now()
	result, err := s.db.Exec(`UPDATE sites SET name=?, domains=?, upstream=?, enabled=?, domain=?, backend_url=?, mode=?, maintenance_mode=?, updated_at=? WHERE rowid=?`,
		site.Domain, string(domains), site.BackendURL, site.Enabled, site.Domain, site.BackendURL, site.Mode, site.Mode == "emergency", now, id)
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, nil
	}
	site.ID, site.UpdatedAt = id, now
	return &site, nil
}

func (s *Store) DeleteV2Site(id int64) error {
	if id <= 0 {
		return fmt.Errorf("site ID must be positive")
	}
	_, err := s.db.Exec("DELETE FROM sites WHERE rowid = ?", id)
	return err
}

func normalizeV2Domain(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimPrefix(value, "https://")
	return strings.TrimSuffix(value, "/")
}

func validV2Mode(mode string) bool {
	switch mode {
	case "monitor", "protect", "emergency":
		return true
	default:
		return false
	}
}
