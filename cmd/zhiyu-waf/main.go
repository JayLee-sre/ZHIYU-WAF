package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"

	"zhiyuwaf/internal/acme"
	"zhiyuwaf/internal/ai"
	"zhiyuwaf/internal/ai/openai"
	"zhiyuwaf/internal/alert"
	"zhiyuwaf/internal/config"
	"zhiyuwaf/internal/core"
	"zhiyuwaf/internal/dashboard"
	"zhiyuwaf/internal/engine"
	"zhiyuwaf/internal/firewall/nftables"
	"zhiyuwaf/internal/geo"
	"zhiyuwaf/internal/model"
	"zhiyuwaf/internal/proxy"
	"zhiyuwaf/internal/sshmon"
	"zhiyuwaf/internal/store"
	"zhiyuwaf/internal/threatintel"
	v2service "zhiyuwaf/internal/v2"
)

func main() {
	configPath := flag.String("config", "configs/zhiyu-waf.yaml", "path to config file")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config validation failed: %v", err)
	}
	log.Printf("config loaded from %s", *configPath)

	applyEnvOverrides(cfg)

	// Initialize store
	var dbStore store.Storage
	switch cfg.Storage.Type {
	case "mysql":
		if cfg.Storage.DSN == "" {
			log.Fatal("MySQL selected but storage.dsn is empty")
		}
		dbStore, err = store.NewMySQLStore(cfg.Storage.DSN, cfg.Storage.MaxOpenConns, cfg.Storage.MaxIdleConns)
	default:
		dbStore, err = store.NewStore(cfg.Storage.Path)
	}
	if err != nil {
		log.Fatalf("failed to init store: %v", err)
	}
	defer dbStore.Close()
	log.Printf("database initialized at %s", cfg.Storage.Path)

	// Init geo rules table
	if err := dbStore.InitGeoTable(); err != nil {
		log.Printf("warning: failed to init geo table: %v", err)
	}

	// First-time setup: generate one-time password if none set
	if storedHash, _ := dbStore.GetSetting("admin_password_hash"); storedHash == "" {
		otp := generateOTP()
		hash, _ := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
		dbStore.SetSetting("admin_password_hash", string(hash))
		log.Printf("=======================================================")
		log.Printf("首次启动 - 管理员初始密码: %s", otp)
		log.Printf("请登录后立即修改密码！")
		log.Printf("=======================================================")
	}

	// Initialize GeoIP resolver
	geoResolver := geo.NewResolver()

	// Initialize the V2 nftables manager. Failure degrades only kernel-level
	// enforcement; application-layer WAF decisions continue to protect traffic.
	hostFirewall := nftables.New()
	if err := hostFirewall.Sync(context.Background()); err != nil {
		log.Printf("warning: nftables firewall degraded: %v", err)
	}
	if sqliteStore, ok := dbStore.(*store.Store); ok {
		replayV2Blocklist(sqliteStore, hostFirewall)
	}

	// Initialize SSH monitor through the V2 Firewall interface.
	sshMonitor := sshmon.New(sshmon.Config{
		Enabled:    cfg.SSH.Enabled,
		LogPath:    cfg.SSH.LogPath,
		MaxFails:   cfg.SSH.MaxFails,
		BanMinutes: cfg.SSH.BanMinutes,
	}, dbStore, geoResolver, hostFirewall)
	sshMonitor.Start()
	defer sshMonitor.Stop()

	// Create dashboard server and load persisted AI settings
	dashServer := dashboard.NewServer(cfg, dbStore)
	dashServer.SetConfigPath(*configPath)
	dashServer.FirewallStatusProvider = func() core.FirewallStatus { return hostFirewall.Status(context.Background()) }
	dashServer.FirewallBlock = func(ip net.IP, ttl time.Duration, reason string) error {
		return hostFirewall.BlockIP(context.Background(), ip, ttl, reason)
	}
	dashServer.FirewallUnblock = func(ip net.IP) error {
		return hostFirewall.UnblockIP(context.Background(), ip)
	}
	dashServer.LoadAISettingsFromDB()
	applyEnvOverrides(cfg)

	// Load rules from YAML files
	ruleSet := engine.NewRuleSet()
	ruleSet.SetPreset(cfg.Engine.Preset)
	if err := ruleSet.LoadFromDir(cfg.Engine.RulesDir); err != nil {
		log.Fatalf("failed to load rules: %v", err)
	}

	// Also load rules from DB
	dbRules, err := dbStore.ListRules()
	if err != nil {
		log.Printf("warning: failed to load DB rules: %v", err)
	} else {
		ruleSet.LoadFromDB(dbRules)
		log.Printf("loaded %d rules from database", len(dbRules))
	}

	// Create detection pipeline
	pipeline := engine.NewPipeline(ruleSet, cfg.Engine.RateLimit.RequestsPerMinute, cfg.Engine.RateLimit.BurstSize)
	pipeline.SetObservationMode(cfg.Engine.ObservationMode)
	if cfg.Engine.ObservationMode {
		log.Println("WARNING: observation mode enabled — requests will NOT be blocked, only logged")
	}

	// Load IP lists
	whitelist, _ := dbStore.GetIPListMap("whitelist")
	blacklist, _ := dbStore.GetIPListMap("blacklist")
	pipeline.UpdateIPLists(whitelist, blacklist)

	// V2 owns the request decision path; the legacy pipeline remains available
	// only as a migration fallback inside the proxy handler.
	v2Pipeline, err := v2service.New(v2service.Config{
		RequestsPerMinute: cfg.Engine.RateLimit.RequestsPerMinute,
		BurstSize:         cfg.Engine.RateLimit.BurstSize,
	}, ruleSet)
	if err != nil {
		log.Fatalf("failed to initialize V2 pipeline: %v", err)
	}
	defer v2Pipeline.Close()
	v2Pipeline.UpdateIPLists(whitelist, blacklist)

	// Set geo resolver for geo-blocking
	pipeline.SetGeoResolver(geoResolver)
	if blocked, err := dbStore.GetBlockedCountries(); err == nil {
		pipeline.UpdateGeoRules(blocked)
		log.Printf("loaded %d geo-blocked countries", len(blocked))
	}

	// Initialize AI analyzer (if enabled)
	var currentAI ai.Analyzer
	currentAI = initAI(cfg, pipeline, dashServer.IncrementAIUsage)
	if currentAI != nil {
		currentAI.SetAllowedCheck(dashServer.IsCommunityAIAllowed)
	}

	var handler *proxy.Handler
	siteResolver := proxy.NewMemorySiteResolver(loadEnabledSites(dbStore))

	// Wire up callbacks for hot-reload
	dashServer.OnAIConfigChanged = func() {
		log.Println("AI config changed, reinitializing analyzer...")
		if currentAI != nil {
			currentAI.Stop()
		}
		currentAI = initAI(cfg, pipeline, dashServer.IncrementAIUsage)
		if currentAI != nil {
			currentAI.SetAllowedCheck(dashServer.IsCommunityAIAllowed)
		}
	}
	dashServer.OnConfigReload = func() {
		log.Println("config reload requested")
		newCfg, err := config.Load(*configPath)
		if err != nil {
			log.Printf("reload config failed: %v", err)
			return
		}
		newRuleSet := engine.NewRuleSet()
		newRuleSet.SetPreset(newCfg.Engine.Preset)
		if err := newRuleSet.LoadFromDir(newCfg.Engine.RulesDir); err != nil {
			log.Printf("reload rules failed: %v", err)
			return
		}
		dbRules, _ := dbStore.ListRules()
		newRuleSet.LoadFromDB(dbRules)
		pipeline.UpdateRules(newRuleSet)
		v2Pipeline.UpdateRules(newRuleSet)
		// Update handler config to avoid stale references
		handler.UpdateConfig(newCfg.Proxy.BackendAddr, newCfg.Proxy.ReadTimeout, newCfg.Proxy.WriteTimeout, newCfg.Proxy.DynamicProtect)
		handler.SetTrustedProxies(newCfg.Proxy.TrustedProxies)
		log.Println("config reloaded")
	}
	dashServer.OnIPListChanged = func() {
		whitelist, _ := dbStore.GetIPListMap("whitelist")
		blacklist, _ := dbStore.GetIPListMap("blacklist")
		pipeline.UpdateIPLists(whitelist, blacklist)
		v2Pipeline.UpdateIPLists(whitelist, blacklist)
		log.Println("IP lists reloaded")
	}
	dashServer.OnSitesChanged = func() {
		siteResolver.Update(loadEnabledSites(dbStore))
		log.Println("sites reloaded")
	}
	dashServer.OnRulesChanged = func() {
		activeCfg, err := config.Load(*configPath)
		if err != nil {
			log.Printf("reload rules failed: %v", err)
			return
		}
		applyEnvOverrides(activeCfg)
		newRuleSet := engine.NewRuleSet()
		newRuleSet.SetPreset(activeCfg.Engine.Preset)
		if err := newRuleSet.LoadFromDir(activeCfg.Engine.RulesDir); err != nil {
			log.Printf("reload rules failed: %v", err)
			return
		}
		dbRules, _ := dbStore.ListRules()
		newRuleSet.LoadFromDB(dbRules)
		pipeline.UpdateRules(newRuleSet)
		log.Println("rules reloaded")
	}
	dashServer.OnGeoRulesChanged = func() {
		if blocked, err := dbStore.GetBlockedCountries(); err == nil {
			pipeline.UpdateGeoRules(blocked)
			log.Println("geo rules reloaded")
		}
	}

	// Initialize threat intelligence syncer
	var threatSyncer *threatintel.Syncer
	threatSyncer = setupThreatIntel(cfg, dbStore, func() {
		whitelist, _ := dbStore.GetIPListMap("whitelist")
		blacklist, _ := dbStore.GetIPListMap("blacklist")
		pipeline.UpdateIPLists(whitelist, blacklist)
		log.Println("threat intel IPs synced to blacklist")
	})
	if threatSyncer != nil {
		dashServer.ThreatSyncerStatus = func() (time.Time, int) { return threatSyncer.Status() }
		dashServer.ThreatSyncerSync = func() { threatSyncer.Sync() }
	}
	dashServer.OnThreatIntelChanged = func() {
		log.Println("threat intel config changed, reinitializing...")
		if threatSyncer != nil {
			threatSyncer.Stop()
		}
		threatSyncer = setupThreatIntel(cfg, dbStore, func() {
			whitelist, _ := dbStore.GetIPListMap("whitelist")
			blacklist, _ := dbStore.GetIPListMap("blacklist")
			pipeline.UpdateIPLists(whitelist, blacklist)
			log.Println("threat intel IPs synced to blacklist")
		})
		if threatSyncer != nil {
			dashServer.ThreatSyncerStatus = func() (time.Time, int) { return threatSyncer.Status() }
			dashServer.ThreatSyncerSync = func() { threatSyncer.Sync() }
			// Trigger immediate sync after config change
			go threatSyncer.Sync()
		} else {
			dashServer.ThreatSyncerStatus = nil
			dashServer.ThreatSyncerSync = nil
		}
	}

	// Initialize alert channels
	var alerters []alert.Alerter
	if cfg.Alert.Enabled {
		if cfg.Alert.WebhookURL != "" {
			alerters = append(alerters, alert.NewWebhookAlerter(cfg.Alert.WebhookURL, cfg.Alert.ThrottleMin))
			log.Printf("alert webhook configured: %s", cfg.Alert.WebhookURL)
		}
		if cfg.Alert.Email.Host != "" && len(cfg.Alert.Email.To) > 0 {
			alerters = append(alerters, alert.NewEmailAlerter(
				cfg.Alert.Email.Host, cfg.Alert.Email.Port,
				cfg.Alert.Email.Username, cfg.Alert.Email.Password,
				cfg.Alert.Email.From, cfg.Alert.Email.To,
				cfg.Alert.ThrottleMin,
			))
			log.Printf("alert email configured: %s", cfg.Alert.Email.Host)
		}
	}

	// Start background log writer + WebSocket broadcast
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for logEntry := range pipeline.AttackLogChan() {
			// Resolve GeoIP region for the attacker
			logEntry.Region = geoResolver.FormatRegion(logEntry.ClientIP)
			if err := dbStore.InsertAttackLog(logEntry); err != nil {
				log.Printf("failed to save attack log: %v", err)
			}
			if logEntry.Source == "ai" {
				if err := dbStore.InsertAuditEvent(model.AuditEvent{
					ID:        "ai-" + logEntry.ID,
					Timestamp: time.Now(),
					Actor:     "ai",
					ClientIP:  logEntry.ClientIP,
					Action:    "ai_block",
					Status:    "blocked",
					Detail:    logEntry.RuleName + ": " + logEntry.AIReasoning,
				}); err != nil {
					log.Printf("failed to save AI audit event: %v", err)
				}
			}
			// Send alerts for high/critical severity attacks
			if len(alerters) > 0 && (logEntry.Severity == "high" || logEntry.Severity == "critical") {
				a := alert.Alert{
					Title:     "WAF Attack Blocked: " + logEntry.RuleName,
					Severity:  logEntry.Severity,
					Message:   logEntry.Path + " from " + logEntry.ClientIP,
					SourceIP:  logEntry.ClientIP,
					RuleID:    logEntry.RuleID,
					Timestamp: logEntry.Timestamp,
				}
				for _, alerter := range alerters {
					go alerter.Send(a)
				}
			}
			dashServer.Hub().Broadcast(logEntry)
		}
	}()

	// Periodic log cleanup (daily)
	if cfg.Storage.LogRetentionDays > 0 {
		go func() {
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					dbStore.CleanupOldLogs(cfg.Storage.LogRetentionDays)
				case <-ctx.Done():
					return
				}
			}
		}()
		go dbStore.CleanupOldLogs(cfg.Storage.LogRetentionDays)
	}

	// Start config hot-reload (file watcher)
	config.Watch(*configPath, func(newCfg *config.Config) {
		log.Println("config file changed, reloading...")
		newRuleSet := engine.NewRuleSet()
		newRuleSet.SetPreset(newCfg.Engine.Preset)
		if err := newRuleSet.LoadFromDir(newCfg.Engine.RulesDir); err != nil {
			log.Printf("reload rules failed: %v", err)
			return
		}
		dbRules, _ := dbStore.ListRules()
		newRuleSet.LoadFromDB(dbRules)
		pipeline.UpdateRules(newRuleSet)
		v2Pipeline.UpdateRules(newRuleSet)
		// Update handler config to avoid stale references
		handler.UpdateConfig(newCfg.Proxy.BackendAddr, newCfg.Proxy.ReadTimeout, newCfg.Proxy.WriteTimeout, newCfg.Proxy.DynamicProtect)
		handler.SetTrustedProxies(newCfg.Proxy.TrustedProxies)
	})

	// Create proxy handler and listener. V2 uses explicit reverse-proxy
	// deployment; nftables is reserved for managed blocklist enforcement.
	handler = proxy.NewHandler(cfg.Proxy.BackendAddr, pipeline, cfg.Proxy.ReadTimeout, cfg.Proxy.WriteTimeout)
	handler.SetSiteResolver(siteResolver)
	handler.SetDynamicProtect(cfg.Proxy.DynamicProtect)
	handler.SetTrustedProxies(cfg.Proxy.TrustedProxies)
	handler.SetV2Service(v2Pipeline)
	if sqliteStore, ok := dbStore.(*store.Store); ok {
		handler.SetV2DecisionCallback(func(decision *core.Decision, request *core.RequestContext) {
			persistV2Decision(sqliteStore, hostFirewall, decision, request)
		})
	}
	handler.SetMetricsCallbacks(dashboard.IncrementRequests, dashboard.IncrementBlocked)
	listener := proxy.NewListener(cfg.Proxy.ListenAddr, cfg.Proxy.TLSListenAddr, handler, cfg.Proxy.TLSCertFile, cfg.Proxy.TLSKeyFile)

	// Setup ACME if enabled (auto TLS certificates from Let's Encrypt)
	if cfg.Proxy.ACMEEnabled && len(cfg.Proxy.ACMEDomains) > 0 {
		certsDir := filepath.Join(filepath.Dir(cfg.Storage.Path), "certs")
		acmeMgr := acme.New(certsDir, cfg.Proxy.ACMEEmail, cfg.Proxy.ACMEDomains)
		listener.SetACMEManager(acmeMgr)
		log.Printf("ACME enabled for domains: %v", cfg.Proxy.ACMEDomains)
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("received signal %v, shutting down...", sig)
		cancel()
		if currentAI != nil {
			currentAI.Stop()
		}
		pipeline.Close()
		os.Exit(0)
	}()

	// Start dashboard in background
	go func() {
		if err := dashServer.Start(ctx); err != nil {
			log.Printf("dashboard error: %v", err)
		}
	}()

	// Start proxy
	log.Printf("ZhiYu-WAF starting on %s -> %s", cfg.Proxy.ListenAddr, cfg.Proxy.BackendAddr)
	log.Printf("Rules loaded: %d", len(ruleSet.Rules()))
	log.Printf("Dashboard at %s", cfg.Dashboard.ListenAddr)

	if err := listener.Start(ctx); err != nil {
		log.Fatalf("proxy failed: %v", err)
	}
}

func applyEnvOverrides(cfg *config.Config) {
	if envSecret := os.Getenv("ZHIYU_WAF_JWT_SECRET"); envSecret != "" {
		cfg.Dashboard.JWTSecret = envSecret
		log.Printf("JWT secret loaded from environment")
	}
	if apiKey := os.Getenv("ZHIYU_WAF_OPENAI_API_KEY"); apiKey != "" {
		cfg.AI.Providers.OpenAI.APIKey = apiKey
		log.Printf("OpenAI-compatible API key loaded from environment")
	}
	if baseURL := os.Getenv("ZHIYU_WAF_OPENAI_BASE_URL"); baseURL != "" {
		cfg.AI.Providers.OpenAI.BaseURL = baseURL
	}
	if model := os.Getenv("ZHIYU_WAF_OPENAI_MODEL"); model != "" {
		cfg.AI.Providers.OpenAI.Model = model
	}
}

// initAI initializes or reinitializes the AI analyzer based on current config.
func initAI(cfg *config.Config, pipeline *engine.Pipeline, onCall func()) ai.Analyzer {
	if !cfg.AI.Enabled {
		pipeline.SetAIAnalyzer(nil)
		log.Println("AI analyzer disabled")
		return nil
	}

	// Guard: empty API key causes continuous failures → circuit breaker → false blocks
	if cfg.AI.Providers.OpenAI.APIKey == "" {
		pipeline.SetAIAnalyzer(nil)
		log.Println("AI analyzer disabled: API key is empty")
		return nil
	}

	var provider ai.Provider
	switch cfg.AI.Provider {
	case "openai":
		provider = openai.NewClient(
			cfg.AI.Providers.OpenAI.APIKey,
			cfg.AI.Providers.OpenAI.Model,
			cfg.AI.Providers.OpenAI.BaseURL,
		)
	default:
		log.Printf("unknown AI provider: %s, AI disabled", cfg.AI.Provider)
		return nil
	}

	analyzer := ai.NewAnalyzer(
		provider,
		time.Duration(cfg.AI.CacheTTL)*time.Second,
		cfg.AI.MaxRequests,
		cfg.AI.AsyncTimeout,
		cfg.AI.FailOpen,
		cfg.AI.HighRiskPaths,
		cfg.AI.PerIPRate,
		cfg.AI.PerIPBurst,
		cfg.AI.CircuitThreshold,
		cfg.AI.CircuitReset,
	)
	if onCall != nil {
		analyzer.SetOnCall(onCall)
	}
	pipeline.SetAIAnalyzer(analyzer)
	log.Printf("AI analyzer initialized: provider=%s model=%s", cfg.AI.Provider, provider.Name())
	return analyzer
}

func parseAddr(addr string) (string, int, error) {
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(portStr)
	return addr, port, err
}

func loadEnabledSites(dbStore store.Storage) []model.Site {
	sites, err := dbStore.ListEnabledSites()
	if err != nil {
		log.Printf("warning: failed to load sites: %v", err)
		return nil
	}
	log.Printf("loaded %d enabled sites", len(sites))
	return sites
}

func generateOTP() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b) // 24-char hex password
}

func setupThreatIntel(cfg *config.Config, dbStore store.Storage, onChanged func()) *threatintel.Syncer {
	apiKey := os.Getenv("ZHIYU_WAF_ABUSEIPDB_KEY")
	if apiKey == "" {
		if v, _ := dbStore.GetSetting("threatintel_api_key"); v != "" {
			apiKey = v
		}
	}
	if apiKey == "" {
		log.Println("threat intelligence: no API key configured, skipping")
		return nil
	}

	feed := threatintel.NewAbuseIPDB(apiKey)
	syncer := threatintel.NewSyncer(feed, dbStore, onChanged)
	syncer.Start(6 * time.Hour)
	log.Println("threat intelligence: AbuseIPDB syncer started (every 6h)")
	return syncer
}

// persistV2Decision records the privacy-minimized V2 event and only then
// delegates qualifying Block decisions to the dedicated nftables Firewall.
// Persistence or firewall failures are isolated from the active request path.
func persistV2Decision(db *store.Store, firewall core.Firewall, decision *core.Decision, request *core.RequestContext) {
	if db == nil || decision == nil || request == nil || request.ClientIP == nil {
		return
	}
	first := core.Detection{}
	if len(decision.Detections) > 0 {
		first = decision.Detections[0]
	}
	event := model.SecurityEvent{
		RequestID:  request.RequestID,
		SiteID:     request.SiteID,
		ClientIP:   request.ClientIP.String(),
		Method:     request.Method,
		Host:       request.Host,
		Path:       request.Path,
		RuleID:     first.RuleID,
		Category:   first.Category,
		Severity:   first.Severity,
		Confidence: first.Confidence,
		RiskScore:  decision.Risk.Score,
		Action:     string(decision.Action),
		UserAgent:  request.Header.Get("User-Agent"),
		CreatedAt:  time.Now(),
	}
	if err := db.InsertSecurityEvent(event); err != nil {
		log.Printf("failed to persist V2 security event: %v", err)
	}
	if decision.Risk.Score > 0 {
		if err := db.RecordRiskEvent(model.RiskEvent{
			ClientIP:  request.ClientIP.String(),
			Score:     decision.Risk.Score,
			Level:     decision.Risk.Level,
			Reason:    strings.Join(decision.Risk.Reasons, "; "),
			Action:    string(decision.Action),
			CreatedAt: time.Now(),
		}); err != nil {
			log.Printf("failed to persist V2 risk event: %v", err)
		}
	}
	if decision.Action != core.ActionBlock || firewall == nil {
		return
	}
	ttl := v2BlockTTL(decision.Risk.Score)
	expiresAt := time.Now().Add(ttl)
	family := 6
	if request.ClientIP.To4() != nil {
		family = 4
	}
	if err := db.UpsertBlockedIP(model.BlockedIP{
		IP:        request.ClientIP.String(),
		Family:    family,
		Reason:    strings.Join(decision.Risk.Reasons, "; "),
		Score:     decision.Risk.Score,
		ExpiresAt: &expiresAt,
		Source:    "local",
		CreatedAt: time.Now(),
	}); err != nil {
		log.Printf("failed to persist V2 blocklist entry: %v", err)
	}
	if err := firewall.BlockIP(context.Background(), request.ClientIP, ttl, "risk score "+strconv.Itoa(decision.Risk.Score)); err != nil {
		log.Printf("V2 firewall degraded, application block remains active: %v", err)
	}
}

func v2BlockTTL(score int) time.Duration {
	switch {
	case score >= 98:
		return 24 * time.Hour
	case score >= 92:
		return time.Hour
	default:
		return 10 * time.Minute
	}
}

func replayV2Blocklist(db *store.Store, firewall core.Firewall) {
	if db == nil || firewall == nil {
		return
	}
	now := time.Now()
	entries, err := db.ListActiveBlockedIPs(now)
	if err != nil {
		log.Printf("failed to load V2 blocklist for replay: %v", err)
		return
	}
	for _, entry := range entries {
		ip := net.ParseIP(entry.IP)
		if ip == nil {
			log.Printf("skip invalid persisted blocklist IP: %q", entry.IP)
			continue
		}
		// A persistent block uses a zero timeout; temporary blocks use only the
		// remaining duration so a restart never extends the operator's TTL.
		ttl := time.Duration(0)
		if entry.ExpiresAt != nil {
			ttl = time.Until(*entry.ExpiresAt)
			if ttl <= 0 {
				continue
			}
		}
		if err := firewall.BlockIP(context.Background(), ip, ttl, entry.Reason); err != nil {
			log.Printf("failed to replay blocklist IP %s: %v", entry.IP, err)
		}
	}
	if len(entries) > 0 {
		log.Printf("replayed %d active V2 blocklist entries", len(entries))
	}
}
