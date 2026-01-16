package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bluefunda/cai-cli/internal/config"
)

// TokenResponse represents the OAuth2 token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

// DeviceCodeResponse represents the device authorization response
type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// Client handles authentication operations
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewClient creates a new auth client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// StartDeviceAuth initiates the device authorization flow
func (c *Client) StartDeviceAuth() (*DeviceCodeResponse, error) {
	deviceAuthURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/auth/device", c.cfg.AuthURL, c.cfg.Realm)

	data := url.Values{}
	data.Set("client_id", c.cfg.ClientID)
	data.Set("scope", "openid profile email offline_access")

	req, err := http.NewRequest("POST", deviceAuthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device authorization failed: %s", string(body))
	}

	var deviceResp DeviceCodeResponse
	if err := json.Unmarshal(body, &deviceResp); err != nil {
		return nil, fmt.Errorf("failed to parse device response: %w", err)
	}

	return &deviceResp, nil
}

// PollForToken polls the token endpoint waiting for user authorization
func (c *Client) PollForToken(deviceCode string, interval int) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", c.cfg.AuthURL, c.cfg.Realm)

	pollInterval := time.Duration(interval) * time.Second
	if pollInterval < 5*time.Second {
		pollInterval = 5 * time.Second
	}

	for {
		data := url.Values{}
		data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
		data.Set("client_id", c.cfg.ClientID)
		data.Set("device_code", deviceCode)

		req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			var tokenResp TokenResponse
			if err := json.Unmarshal(body, &tokenResp); err != nil {
				return nil, fmt.Errorf("failed to parse token response: %w", err)
			}
			return &tokenResp, nil
		}

		// Check for pending/slow_down errors
		var errResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil {
			switch errResp.Error {
			case "authorization_pending":
				// User hasn't authorized yet, keep polling
				time.Sleep(pollInterval)
				continue
			case "slow_down":
				// Increase polling interval
				pollInterval += 5 * time.Second
				time.Sleep(pollInterval)
				continue
			case "expired_token":
				return nil, fmt.Errorf("device code expired. Please try again")
			case "access_denied":
				return nil, fmt.Errorf("access denied by user")
			default:
				return nil, fmt.Errorf("authorization failed: %s - %s", errResp.Error, errResp.ErrorDescription)
			}
		}

		return nil, fmt.Errorf("unexpected response: %s", string(body))
	}
}

// RefreshToken refreshes the access token using a refresh token
func (c *Client) RefreshToken(refreshToken string) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", c.cfg.AuthURL, c.cfg.Realm)

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", c.cfg.ClientID)
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: %s", string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// Logout invalidates the current session
func (c *Client) Logout(refreshToken string) error {
	logoutURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/logout", c.cfg.AuthURL, c.cfg.Realm)

	data := url.Values{}
	data.Set("client_id", c.cfg.ClientID)
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", logoutURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("logout failed: %s", string(body))
	}

	return nil
}
