package cmd

import (
	"github.com/spf13/cobra"

	"github.com/bluefunda/cai-cli/internal/auth"
	"github.com/bluefunda/cai-cli/internal/ui"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in via browser (opens Keycloak)",
	RunE:  runLogin,
}

func runLogin(cmd *cobra.Command, args []string) error {
	cfg := loadConfig()

	tok, err := auth.LoginWithDevice(cfg.Domain)
	if err != nil {
		ui.Error("Login failed: " + err.Error())
		return err
	}

	if err := saveAuthTokens(cfg, tok); err != nil {
		ui.Error("Failed to save config: " + err.Error())
		return err
	}

	ui.Success("Logged in successfully")
	return nil
}
