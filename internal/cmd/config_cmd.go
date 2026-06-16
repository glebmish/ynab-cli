package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebmish/ynab-cli/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := configFilePath()
		if err != nil {
			return err
		}

		// Preserve existing values if the file is already there.
		existing, _ := config.Load(cfgPath)
		if existing == nil {
			existing = &config.Config{}
		}
		existing.ApplyEnv()

		reader := bufio.NewReader(os.Stdin)

		fmt.Fprint(os.Stderr, "Access token")
		if existing.AccessToken != "" {
			fmt.Fprintf(os.Stderr, " [%s]", maskToken(existing.AccessToken))
		}
		fmt.Fprint(os.Stderr, ": ")
		token, _ := reader.ReadString('\n')
		token = strings.TrimSpace(token)
		if token == "" {
			token = existing.AccessToken
		}
		if token == "" {
			return fmt.Errorf("access token is required\n  Get one at https://app.ynab.com/settings/developer")
		}

		defaultPlan := existing.PlanID
		if defaultPlan == "" {
			defaultPlan = "last-used"
		}
		fmt.Fprintf(os.Stderr, "Plan ID [%s]: ", defaultPlan)
		planID, _ := reader.ReadString('\n')
		planID = strings.TrimSpace(planID)
		if planID == "" {
			planID = defaultPlan
		}

		content := fmt.Sprintf("access_token: %q\nplan_id: %q\nbase_url: \"https://api.ynab.com/v1\"\n", token, planID)

		dir := filepath.Dir(cfgPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating config directory: %w", err)
		}

		if err := os.WriteFile(cfgPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Config written to %s\n", cfgPath)
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the effective config file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := configFilePath()
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the resolved config (token masked unless --unmasked)",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := configFilePath()
		if err != nil {
			return err
		}
		cfg, err := config.Load(path)
		if err != nil {
			return err
		}
		cfg.ApplyEnv()
		unmasked, _ := cmd.Flags().GetBool("unmasked")
		token := maskToken(cfg.AccessToken)
		if unmasked {
			token = cfg.AccessToken
		}
		fmt.Printf("path:         %s\n", path)
		fmt.Printf("access_token: %s\n", token)
		fmt.Printf("plan_id:      %s\n", cfg.PlanID)
		fmt.Printf("base_url:     %s\n", cfg.BaseURL)
		return nil
	},
}

func configFilePath() (string, error) {
	if v := os.Getenv("YNAB_CONFIG"); v != "" {
		return v, nil
	}
	return config.DefaultPath()
}

func maskToken(t string) string {
	if t == "" {
		return "(unset)"
	}
	if len(t) <= 8 {
		return strings.Repeat("*", len(t))
	}
	return t[:4] + strings.Repeat("*", len(t)-8) + t[len(t)-4:]
}

func init() {
	configShowCmd.Flags().Bool("unmasked", false, "Print the token in clear text (for headless bootstrap)")
	configCmd.AddCommand(configInitCmd, configPathCmd, configShowCmd)
	rootCmd.AddCommand(configCmd)
}
