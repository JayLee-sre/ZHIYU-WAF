package dashboard

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"zhiyuwaf/internal/config"
	"zhiyuwaf/internal/core"
	"zhiyuwaf/internal/store"
)

type Server struct {
	cfg        *config.Config
	configPath string
	store      store.Storage
	hub        *Hub
	server     *http.Server

	// Build and GitHub Release update-check state. The checker is intentionally
	// unauthenticated because the upstream repository is public; this avoids
	// storing GitHub credentials in the control plane.
	buildVersion       string
	githubReleaseURL   string
	githubHTTPClient   *http.Client
	updateCacheMu      sync.Mutex
	updateCache        updateCheckResponse
	updateCacheExpires time.Time

	// Callbacks for hot-reload (set by main.go)
	OnAIConfigChanged    func()
	OnConfigReload       func()
	OnIPListChanged      func()
	OnSitesChanged       func()
	OnRulesChanged       func()
	OnGeoRulesChanged    func()
	OnThreatIntelChanged func()
	OnCertReload         func()

	// Threat intel syncer
	ThreatSyncerStatus     func() (time.Time, int)
	ThreatSyncerSync       func()
	FirewallStatusProvider func() core.FirewallStatus
	FirewallBlock          func(net.IP, time.Duration, string) error
	FirewallUnblock        func(net.IP) error
}

func NewServer(cfg *config.Config, s store.Storage) *Server {
	srv := &Server{
		cfg:              cfg,
		store:            s,
		hub:              NewHub(cfg.Dashboard.CORSOrigins),
		buildVersion:     BuildVersion,
		githubReleaseURL: GitHubLatestReleaseURL,
		githubHTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	srv.server = &http.Server{
		Addr:      cfg.Dashboard.ListenAddr,
		Handler:   srv.setupRouter(),
		TLSConfig: nil,
	}

	// Configure TLS if cert/key provided
	if cfg.Dashboard.TLSCertFile != "" && cfg.Dashboard.TLSKeyFile != "" {
		srv.server.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return srv
}

func (s *Server) Start(ctx context.Context) error {
	go s.hub.Run()

	go func() {
		<-ctx.Done()
		s.hub.Stop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.server.Shutdown(shutdownCtx)
	}()

	log.Printf("dashboard listening on %s", s.cfg.Dashboard.ListenAddr)

	if s.cfg.Dashboard.TLSCertFile != "" && s.cfg.Dashboard.TLSKeyFile != "" {
		log.Printf("dashboard TLS enabled")
		if err := s.server.ListenAndServeTLS(s.cfg.Dashboard.TLSCertFile, s.cfg.Dashboard.TLSKeyFile); err != http.ErrServerClosed {
			return err
		}
	} else {
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			return err
		}
	}
	return nil
}

// FirewallStatus returns an explicit degraded value until the V2 nftables
// manager has been wired by the application bootstrap.
func (s *Server) FirewallStatus() core.FirewallStatus {
	if s.FirewallStatusProvider != nil {
		return s.FirewallStatusProvider()
	}
	return core.FirewallStatus{Degraded: true, Message: "firewall status unavailable"}
}

func (s *Server) Hub() *Hub {
	return s.hub
}

func (s *Server) SetConfigPath(path string) {
	s.configPath = path
}

// SetBuildVersion assigns the current build's semantic version. It is used
// only for comparing against public GitHub Release tags.
func (s *Server) SetBuildVersion(version string) {
	if version != "" {
		s.buildVersion = version
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.Dashboard.JWTSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	s.hub.HandleWS(w, r)
}
