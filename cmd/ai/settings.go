package ai

import (
	"encoding/json"
	"fmt"

	"github.com/bluefunda/cai-cli/internal/api"
	"github.com/spf13/cobra"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "View user settings",
	Long:  `Display your user settings and preferences.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := RequireAuth(); err != nil {
			return err
		}

		client := api.NewClient(cfg)
		settings, err := client.GetSettings()
		if err != nil {
			return err
		}

		// Pretty print settings
		output, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format settings: %w", err)
		}

		fmt.Println("User Settings:")
		fmt.Println(string(output))

		return nil
	},
}

var rateLimitCmd = &cobra.Command{
	Use:   "rate-limit",
	Short: "View rate limit status",
	Long:  `Display your current rate limit status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := RequireAuth(); err != nil {
			return err
		}

		client := api.NewClient(cfg)
		rateLimit, err := client.GetRateLimit()
		if err != nil {
			return err
		}

		output, err := json.MarshalIndent(rateLimit, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format rate limit: %w", err)
		}

		fmt.Println("Rate Limit Status:")
		fmt.Println(string(output))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(settingsCmd)
	rootCmd.AddCommand(rateLimitCmd)
}
