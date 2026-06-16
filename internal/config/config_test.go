package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `access_token: mytoken
plan_id: plan123
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AccessToken != "mytoken" {
		t.Errorf("AccessToken = %q", cfg.AccessToken)
	}
	if cfg.PlanID != "plan123" {
		t.Errorf("PlanID = %q", cfg.PlanID)
	}
	if cfg.BaseURL != "https://api.ynab.com/v1" {
		t.Errorf("BaseURL = %q", cfg.BaseURL)
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AccessToken != "" {
		t.Errorf("AccessToken = %q, want empty", cfg.AccessToken)
	}
	if cfg.PlanID != "last-used" {
		t.Errorf("PlanID = %q, want last-used", cfg.PlanID)
	}
	if cfg.BaseURL != "https://api.ynab.com/v1" {
		t.Errorf("BaseURL = %q", cfg.BaseURL)
	}
}

func TestEnvVarOverridesFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `access_token: filetoken
plan_id: fileplan
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("YNAB_ACCESS_TOKEN", "envtoken")
	t.Setenv("YNAB_PLAN_ID", "envplan")
	t.Setenv("YNAB_BASE_URL", "https://env.example.com")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	cfg.ApplyEnv()

	if cfg.AccessToken != "envtoken" {
		t.Errorf("AccessToken = %q", cfg.AccessToken)
	}
	if cfg.PlanID != "envplan" {
		t.Errorf("PlanID = %q", cfg.PlanID)
	}
	if cfg.BaseURL != "https://env.example.com" {
		t.Errorf("BaseURL = %q", cfg.BaseURL)
	}
}

func TestApplyFlags(t *testing.T) {
	cfg := &Config{
		AccessToken: "orig",
		PlanID:      "origplan",
		BaseURL:     "https://orig.example.com",
	}
	cfg.ApplyFlags("flagtoken", "flagplan", "https://flag.example.com")
	if cfg.AccessToken != "flagtoken" {
		t.Errorf("AccessToken = %q", cfg.AccessToken)
	}
	if cfg.PlanID != "flagplan" {
		t.Errorf("PlanID = %q", cfg.PlanID)
	}
	if cfg.BaseURL != "https://flag.example.com" {
		t.Errorf("BaseURL = %q", cfg.BaseURL)
	}
}

func TestApplyFlagsEmptyDoesNotOverride(t *testing.T) {
	cfg := &Config{AccessToken: "orig", PlanID: "origplan", BaseURL: "https://orig.example.com"}
	cfg.ApplyFlags("", "", "")
	if cfg.AccessToken != "orig" || cfg.PlanID != "origplan" || cfg.BaseURL != "https://orig.example.com" {
		t.Errorf("flags overrode with empty: %+v", cfg)
	}
}

func TestValidateMissingAccessToken(t *testing.T) {
	cfg := &Config{PlanID: "last-used", BaseURL: "https://api.ynab.com/v1"}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error")
	}
}

func TestValidateOK(t *testing.T) {
	cfg := &Config{AccessToken: "tok", PlanID: "last-used", BaseURL: "https://api.ynab.com/v1"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidateRejectsInsecureBaseURL(t *testing.T) {
	cases := []string{
		"http://api.ynab.com/v1", // plaintext to a real host: token would leak
		"http://evil.example/v1",
		"ftp://api.ynab.com/v1",
		"not-a-url",
	}
	for _, raw := range cases {
		cfg := &Config{AccessToken: "tok", PlanID: "last-used", BaseURL: raw}
		if err := cfg.Validate(); err == nil {
			t.Errorf("Validate() with base_url %q = nil, want error", raw)
		}
	}
}

func TestValidateAllowsLocalhostHTTP(t *testing.T) {
	for _, raw := range []string{"http://localhost:8080/v1", "http://127.0.0.1:9999/v1"} {
		cfg := &Config{AccessToken: "tok", PlanID: "last-used", BaseURL: raw}
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() with base_url %q = %v, want nil (localhost http allowed)", raw, err)
		}
	}
}
