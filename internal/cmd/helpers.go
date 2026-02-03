package cmd

import (
	"fmt"
	"os"

	"github.com/bluefunda/cai-cli/internal/auth"
	"github.com/bluefunda/cai-cli/internal/config"
	caigrpc "github.com/bluefunda/cai-cli/internal/grpc"
	"github.com/bluefunda/cai-cli/internal/ui"
)

// bffConn establishes an authenticated gRPC connection to the BFF.
// Caller must defer conn.Close().
func bffConn() (*caigrpc.Conn, *config.Config, error) {
	cfg := loadConfig()
	if cfg.Auth.AccessToken == "" {
		return nil, cfg, fmt.Errorf("not authenticated; run 'ai login'")
	}

	refreshFunc := func() (string, error) {
		tok, err := auth.Refresh(cfg.Domain, cfg.Auth.RefreshToken)
		if err != nil {
			return "", fmt.Errorf("token refresh failed (run 'ai login'): %w", err)
		}
		cfg.Auth.AccessToken = tok.AccessToken
		cfg.Auth.RefreshToken = tok.RefreshToken
		cfg.Auth.TokenExpiry = tok.Expiry()
		_ = config.Save(cfg)
		return tok.AccessToken, nil
	}

	ts := caigrpc.NewTokenSource(cfg, refreshFunc)
	conn, err := caigrpc.Dial(cfg.BFFURL, ts)
	if err != nil {
		return nil, cfg, err
	}
	return conn, cfg, nil
}

// printer returns a Printer configured from flags and config.
func printer(cfg *config.Config) *ui.Printer {
	return &ui.Printer{
		Out:    os.Stdout,
		Err:    os.Stderr,
		Format: outputFormat(cfg),
	}
}
