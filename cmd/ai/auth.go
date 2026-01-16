package ai

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/bluefunda/cai-cli/internal/auth"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  `Manage authentication with the CAI platform.`,
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to your account",
	Long:  `Authenticate using OAuth device authorization flow. Opens browser for authentication.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		authClient := auth.NewClient(cfg)

		// Start device authorization flow
		fmt.Println("Initiating device authorization...")
		deviceResp, err := authClient.StartDeviceAuth()
		if err != nil {
			return fmt.Errorf("failed to start device authorization: %w", err)
		}

		// Display verification instructions
		fmt.Println()
		fmt.Println("To authenticate, please visit:")
		fmt.Printf("  %s\n", deviceResp.VerificationURI)
		fmt.Println()
		fmt.Printf("And enter code: %s\n", deviceResp.UserCode)
		fmt.Println()

		// Try to open browser automatically
		noBrowser, _ := cmd.Flags().GetBool("no-browser")
		if !noBrowser {
			url := deviceResp.VerificationURIComplete
			if url == "" {
				url = deviceResp.VerificationURI
			}
			if err := openBrowser(url); err == nil {
				fmt.Println("Browser opened automatically.")
			}
		}

		fmt.Println("Waiting for authorization...")
		fmt.Println()

		// Poll for token
		tokenResp, err := authClient.PollForToken(deviceResp.DeviceCode, deviceResp.Interval)
		if err != nil {
			return fmt.Errorf("authorization failed: %w", err)
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

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
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

	loginCmd.Flags().Bool("no-browser", false, "Don't open browser automatically")
}
