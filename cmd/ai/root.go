package ai

import (
	"fmt"

	"github.com/bluefunda/cai-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
	cfgFile string
	cfg     *config.Config
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "ai",
	Short: "CAI - Conversational AI CLI",
	Long: `CAI CLI is a command-line interface for interacting with the
Conversational AI platform. It allows you to chat with AI models,
manage your sessions, and configure your preferences.

Examples:
  ai chat                    Start an interactive chat session
  ai chat list               List all your chat sessions
  ai models list             List available AI models
  ai auth login              Login to your account`,
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cai-cli/config.json)")
	rootCmd.SetVersionTemplate(fmt.Sprintf("CAI CLI version %s\n", version))
}

// GetConfig returns the loaded configuration
func GetConfig() *config.Config {
	return cfg
}

// RequireAuth checks if the user is authenticated
func RequireAuth() error {
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}
	if !cfg.HasToken() {
		return fmt.Errorf("not authenticated. Please run 'ai auth login' first")
	}
	return nil
}
