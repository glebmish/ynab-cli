package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/glebmish/ynab-cli/internal/cliexit"
	"gopkg.in/yaml.v3"
)

type Config struct {
	AccessToken string `yaml:"access_token"`
	PlanID      string `yaml:"plan_id"`
	BaseURL     string `yaml:"base_url"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory (set YNAB_CONFIG to a path): %w", err)
	}
	return filepath.Join(home, ".config", "ynab", "config.yaml"), nil
}

func defaultConfig() *Config {
	return &Config{
		PlanID:  "last-used",
		BaseURL: "https://api.ynab.com/v1",
	}
}

func Load(path string) (*Config, error) {
	cfg := defaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.ynab.com/v1"
	}
	if cfg.PlanID == "" {
		cfg.PlanID = "last-used"
	}
	return cfg, nil
}

func (c *Config) ApplyEnv() {
	if v := os.Getenv("YNAB_ACCESS_TOKEN"); v != "" {
		c.AccessToken = v
	}
	if v := os.Getenv("YNAB_PLAN_ID"); v != "" {
		c.PlanID = v
	}
	if v := os.Getenv("YNAB_BASE_URL"); v != "" {
		c.BaseURL = v
	}
}

func (c *Config) ApplyFlags(accessToken, planID, baseURL string) {
	if accessToken != "" {
		c.AccessToken = accessToken
	}
	if planID != "" {
		c.PlanID = planID
	}
	if baseURL != "" {
		c.BaseURL = baseURL
	}
}

func (c *Config) Validate() error {
	if c.AccessToken == "" {
		path, _ := DefaultPath()
		return &cliexit.AuthError{Err: fmt.Errorf(
			"access_token not configured\n  Set it in %s or YNAB_ACCESS_TOKEN env var\n  Run: ynab config init\n  Get a token at https://app.ynab.com/settings/developer",
			path)}
	}
	if err := validateBaseURL(c.BaseURL); err != nil {
		return &cliexit.ValidationError{Err: err}
	}
	return nil
}

// validateBaseURL ensures the API base URL is https (so the access token is
// never sent in clear text) to a real host. Plain http is allowed only for
// localhost, to support a local mock server in tests/development.
func validateBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("base_url %q is not a valid URL: %w", raw, err)
	}
	if u.Host == "" {
		return fmt.Errorf("base_url %q must be an absolute URL (e.g. https://api.ynab.com/v1)", raw)
	}
	if u.Scheme == "https" {
		return nil
	}
	host := u.Hostname()
	if u.Scheme == "http" && (host == "localhost" || host == "127.0.0.1" || host == "::1") {
		return nil
	}
	return fmt.Errorf("base_url %q must use https (http is allowed only for localhost)", raw)
}
