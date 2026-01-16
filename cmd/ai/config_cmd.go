package ai

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long:  `View and modify CLI configuration settings.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current CLI configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Current Configuration:")
		fmt.Printf("  API Base URL:   %s\n", cfg.APIBaseURL)
		fmt.Printf("  Auth URL:       %s\n", cfg.AuthURL)
		fmt.Printf("  Realm:          %s\n", cfg.Realm)
		fmt.Printf("  Client ID:      %s\n", cfg.ClientID)
		if cfg.HasToken() {
			fmt.Printf("  Authenticated:  Yes\n")
		} else {
			fmt.Printf("  Authenticated:  No\n")
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Long: `Set a configuration value. Available keys:
  api-url     API base URL
  auth-url    Authentication URL
  realm       Keycloak realm
  client-id   OAuth2 client ID`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		switch key {
		case "api-url":
			cfg.APIBaseURL = value
		case "auth-url":
			cfg.AuthURL = value
		case "realm":
			cfg.Realm = value
		case "client-id":
			cfg.ClientID = value
		default:
			return fmt.Errorf("unknown configuration key: %s", key)
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("Configuration updated: %s = %s\n", key, value)
		return nil
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset configuration to defaults",
	Long:  `Reset all configuration values to their defaults.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Print("Are you sure you want to reset configuration? (y/N): ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Reset cancelled.")
				return nil
			}
		}

		cfg.APIBaseURL = "https://api.bluefunda.com/ai"
		cfg.AuthURL = "https://auth.bluefunda.com"
		cfg.Realm = "trm"
		cfg.ClientID = "cai-cli"
		cfg.ClearTokens()

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Println("Configuration reset to defaults.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configResetCmd)

	configResetCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}
