package config

import "testing"

func TestDefaultConfigUsesV2LocalFirstDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Proxy.ListenAddr != ":8080" || cfg.Proxy.BackendAddr != "127.0.0.1:80" {
		t.Fatalf("unexpected proxy defaults: %#v", cfg.Proxy)
	}
	if cfg.AI.Enabled {
		t.Fatal("AI must be disabled by default in V2")
	}
}

func TestValidateGeneratesSecretForPlaceholder(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if cfg.Dashboard.JWTSecret == "ZhiYu-WAF-secret-change-me" || cfg.Dashboard.JWTSecret == "" {
		t.Fatalf("expected generated dashboard secret, got %q", cfg.Dashboard.JWTSecret)
	}
}
