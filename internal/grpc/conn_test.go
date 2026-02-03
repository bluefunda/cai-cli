package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/bluefunda/cai-cli/internal/config"
	"google.golang.org/grpc/metadata"
)

func TestTokenSource_ValidToken(t *testing.T) {
	cfg := &config.Config{
		Auth: config.Auth{
			AccessToken: "valid-token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
		},
	}
	ts := NewTokenSource(cfg, nil)

	token, err := ts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "valid-token" {
		t.Fatalf("expected 'valid-token', got %q", token)
	}
}

func TestTokenSource_ExpiredToken_NoRefresh(t *testing.T) {
	cfg := &config.Config{
		Auth: config.Auth{
			AccessToken: "expired-token",
			TokenExpiry: time.Now().Add(-1 * time.Hour),
		},
	}
	ts := NewTokenSource(cfg, nil)

	// With no refresh func but token exists, it should still return it
	// (the interceptor handles the retry on 401).
	token, err := ts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "expired-token" {
		t.Fatalf("expected 'expired-token', got %q", token)
	}
}

func TestTokenSource_ExpiredToken_WithRefresh(t *testing.T) {
	cfg := &config.Config{
		Auth: config.Auth{
			AccessToken: "expired-token",
			TokenExpiry: time.Now().Add(-1 * time.Hour),
		},
	}
	refreshCalled := false
	refreshFunc := func() (string, error) {
		refreshCalled = true
		cfg.Auth.AccessToken = "refreshed-token"
		cfg.Auth.TokenExpiry = time.Now().Add(1 * time.Hour)
		return "refreshed-token", nil
	}
	ts := NewTokenSource(cfg, refreshFunc)

	token, err := ts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !refreshCalled {
		t.Fatal("refresh func was not called")
	}
	if token != "refreshed-token" {
		t.Fatalf("expected 'refreshed-token', got %q", token)
	}
}

func TestTokenSource_NoToken(t *testing.T) {
	cfg := &config.Config{}
	ts := NewTokenSource(cfg, nil)

	_, err := ts.Token()
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestAttachToken(t *testing.T) {
	cfg := &config.Config{
		Auth: config.Auth{
			AccessToken: "test-token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
		},
	}
	ts := NewTokenSource(cfg, nil)

	ctx, err := attachToken(context.Background(), ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("no outgoing metadata found")
	}

	authValues := md.Get("authorization")
	if len(authValues) == 0 {
		t.Fatal("no authorization metadata found")
	}
	if authValues[0] != "Bearer test-token" {
		t.Fatalf("expected 'Bearer test-token', got %q", authValues[0])
	}
}

func TestDial_EmptyTarget(t *testing.T) {
	cfg := &config.Config{
		Auth: config.Auth{
			AccessToken: "token",
			TokenExpiry: time.Now().Add(1 * time.Hour),
		},
	}
	ts := NewTokenSource(cfg, nil)

	_, err := Dial("", ts)
	if err == nil {
		t.Fatal("expected error for empty target")
	}
}

func TestContextWithTimeout(t *testing.T) {
	ctx, cancel := ContextWithTimeout()
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context to have deadline")
	}
	remaining := time.Until(deadline)
	if remaining < 29*time.Second || remaining > 31*time.Second {
		t.Fatalf("expected ~30s timeout, got %v", remaining)
	}
}
