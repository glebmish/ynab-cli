package cmd

import (
	"fmt"
	"os"

	"github.com/glebmish/ynab-cli/internal/api"
	"github.com/glebmish/ynab-cli/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "ynab",
	Short:        "CLI for the YNAB (You Need A Budget) API",
	Long:         "ynab is a command-line interface for the YNAB API.\nDesigned for AI agents and human operators. 100% API coverage.",
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Offline commands skip config loading.
		switch cmd.Name() {
		case "schema", "skills", "config", "help", "init":
			return nil
		}
		// Walk up to find an offline-group ancestor (e.g. `config init`, `skills list`).
		for p := cmd.Parent(); p != nil; p = p.Parent() {
			switch p.Name() {
			case "schema", "skills", "config":
				return nil
			}
		}

		cfgPath := os.Getenv("YNAB_CONFIG")
		if cfgPath == "" {
			p, err := config.DefaultPath()
			if err != nil {
				return err
			}
			cfgPath = p
		}

		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		cfg.ApplyEnv()

		token, _ := cmd.Flags().GetString("access-token")
		planID, _ := cmd.Flags().GetString("plan-id")
		baseURL, _ := cmd.Flags().GetString("base-url")
		cfg.ApplyFlags(token, planID, baseURL)

		if err := cfg.Validate(); err != nil {
			return err
		}

		client := api.NewClient(cfg.BaseURL, cfg.AccessToken, cfg.PlanID)
		cmd.SetContext(api.WithContext(cmd.Context(), client))
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

// SetVersion wires the build-time version into the root command, enabling
// `ynab --version`. A no-op for empty input so a plain build keeps cobra's
// default (no version flag).
func SetVersion(v string) {
	if v != "" {
		rootCmd.Version = v
	}
}

func init() {
	rootCmd.PersistentFlags().String("format", "json", "Output format: json, ndjson, text")
	rootCmd.PersistentFlags().String("fields", "", "Comma-separated fields to include in output (dotted paths supported, e.g. 'categories.name')")
	rootCmd.PersistentFlags().Bool("dry-run", false, "Show request without executing")
	rootCmd.PersistentFlags().Bool("yes", false, "Skip confirmation prompts")
	rootCmd.PersistentFlags().String("access-token", "", "YNAB access token (overrides config)")
	rootCmd.PersistentFlags().String("plan-id", "", "Plan/budget ID (overrides config)")
	rootCmd.PersistentFlags().String("base-url", "", "API base URL (overrides config)")
	rootCmd.PersistentFlags().String("json", "", "Raw JSON request body for write ops; see 'ynab schema <op>' for the shape")
	rootCmd.PersistentFlags().String("params", "", "Raw JSON object overlaying query/path params, e.g. '{\"account_id\":\"abc\"}'")
}

func confirmDelete(cmd *cobra.Command, resource, id string) error {
	yes, _ := cmd.Flags().GetBool("yes")
	if yes {
		return nil
	}

	fi, err := os.Stdin.Stat()
	if err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		return fmt.Errorf("delete %s %s requires --yes flag in non-interactive mode", resource, id)
	}

	fmt.Fprintf(os.Stderr, "Delete %s %s? [y/N] ", resource, id)
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		return fmt.Errorf("cancelled")
	}
	return nil
}
