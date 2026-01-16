package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	configFileName = "config.json"
	configDirName  = ".cai-cli"
)

// Config holds the CLI configuration
type Config struct {
	APIBaseURL   string `json:"api_base_url"`
	AuthURL      string `json:"auth_url"`
	Realm        string `json:"realm"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		APIBaseURL: "https://api.bluefunda.com/ai",
		AuthURL:    "https://auth.bluefunda.com",
		Realm:      "trm",
		ClientID:   "cai-cli",
	}
}

// configDir returns the path to the config directory
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirName), nil
}

// configPath returns the path to the config file
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// Load loads the configuration from disk
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save saves the configuration to disk
func (c *Config) Save() error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// ClearTokens removes stored tokens
func (c *Config) ClearTokens() {
	c.AccessToken = ""
	c.RefreshToken = ""
}

// HasToken returns true if an access token is stored
func (c *Config) HasToken() bool {
	return c.AccessToken != ""
}
