package cmd

import (
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check gateway health (HTTP)",
	RunE:  runHealth,
}

func runHealth(cmd *cobra.Command, args []string) error {
	cfg := loadConfig()
	if cfg.GatewayURL == "" {
		return fmt.Errorf("gateway_url not configured; run 'ai login' or pass --gateway")
	}

	resp, err := http.Get(cfg.GatewayURL + "/health")
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		fmt.Fprintf(cmd.ErrOrStderr(), "Gateway healthy\n")
	} else {
		fmt.Fprintf(cmd.ErrOrStderr(), "Gateway returned HTTP %d\n", resp.StatusCode)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(body))
	return nil
}
