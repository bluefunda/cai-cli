package ai

import (
	"fmt"
	"syscall"

	"github.com/bluefunda/cai-cli/internal/auth"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  `Manage authentication with the CAI platform.`,
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to your account",
	Long:  `Authenticate with your username and password to access the CAI platform.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		if username == "" {
			fmt.Print("Username: ")
			fmt.Scanln(&username)
		}

		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()
		password := string(passwordBytes)

		authClient := auth.NewClient(cfg)
		tokenResp, err := authClient.LoginWithPassword(username, password)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		cfg.AccessToken = tokenResp.AccessToken
		cfg.RefreshToken = tokenResp.RefreshToken

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		fmt.Println("Login successful!")
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from your account",
	Long:  `Clear stored credentials and logout from the CAI platform.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.RefreshToken != "" {
			authClient := auth.NewClient(cfg)
			if err := authClient.Logout(cfg.RefreshToken); err != nil {
				fmt.Printf("Warning: server logout failed: %v\n", err)
			}
		}

		cfg.ClearTokens()
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to clear credentials: %w", err)
		}

		fmt.Println("Logged out successfully!")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Long:  `Display the current authentication status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.HasToken() {
			fmt.Println("Status: Authenticated")
			fmt.Printf("API URL: %s\n", cfg.APIBaseURL)
			fmt.Printf("Auth URL: %s\n", cfg.AuthURL)
			fmt.Printf("Realm: %s\n", cfg.Realm)
		} else {
			fmt.Println("Status: Not authenticated")
			fmt.Println("Run 'ai auth login' to authenticate")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)

	loginCmd.Flags().StringP("username", "u", "", "Username for login")
}
