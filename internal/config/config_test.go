package config

import (
	"testing"
	"time"
)

func TestTokenValid_Valid(t *testing.T) {
	cfg := &Config{
		Auth: Auth{
			AccessToken: "token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
		},
	}
	if !cfg.TokenValid() {
		t.Error("expected token to be valid")
	}
}

func TestTokenValid_Expired(t *testing.T) {
	cfg := &Config{
		Auth: Auth{
			AccessToken: "token",
			TokenExpiry: time.Now().Add(-1 * time.Hour),
		},
	}
	if cfg.TokenValid() {
		t.Error("expected token to be invalid (expired)")
	}
}

func TestTokenValid_Empty(t *testing.T) {
	cfg := &Config{}
	if cfg.TokenValid() {
		t.Error("expected token to be invalid (empty)")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.GatewayURL != DefaultGatewayURL {
		t.Errorf("expected gateway %s, got %s", DefaultGatewayURL, cfg.GatewayURL)
	}
	if cfg.BFFURL != DefaultBFFURL {
		t.Errorf("expected bff %s, got %s", DefaultBFFURL, cfg.BFFURL)
	}
	if cfg.Domain != DefaultDomain {
		t.Errorf("expected domain %s, got %s", DefaultDomain, cfg.Domain)
	}
	if cfg.Defaults.Model != "openai" {
		t.Errorf("expected model openai, got %s", cfg.Defaults.Model)
	}
	if cfg.Defaults.Output != "text" {
		t.Errorf("expected output text, got %s", cfg.Defaults.Output)
	}
}

func TestAuthURL(t *testing.T) {
	url := AuthURL("example.com")
	expected := "https://auth.example.com/realms/trm/protocol/openid-connect"
	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}
