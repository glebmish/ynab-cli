package cmd

import (
	"fmt"

	"github.com/glebmish/ynab-cli/internal/validate"
	"github.com/spf13/cobra"
)

var plansCmd = &cobra.Command{
	Use:   "plans",
	Short: "Budgets (plans) — list, get, settings",
}

func init() {
	plansCmd.AddCommand(
		newPlansListCmd(),
		newPlansGetCmd(),
		newPlansGetSettingsCmd(),
	)
	rootCmd.AddCommand(plansCmd)
}

func newPlansListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all plans (budgets)",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]string{}
			if v, _ := cmd.Flags().GetBool("include-accounts"); v {
				params["include_accounts"] = "true"
			}
			return doGet(cmd, "/plans", params)
		},
	}
	cmd.Flags().Bool("include-accounts", false, "Include accounts in response")
	return cmd
}

func newPlansGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a plan by ID (uses config plan_id; override with --plan-id)",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]string{}
			if v, _ := cmd.Flags().GetInt64("last-knowledge-of-server"); v > 0 {
				params["last_knowledge_of_server"] = fmt.Sprintf("%d", v)
			}
			return doGet(cmd, "/plans/{plan_id}", params)
		},
	}
	cmd.Flags().Int64("last-knowledge-of-server", 0, "Server knowledge delta")
	return cmd
}

func newPlansGetSettingsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-settings",
		Short: "Get plan settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return doGet(cmd, "/plans/{plan_id}/settings", nil)
		},
	}
}

// ensure validate is referenced (helpers later will use it).
var _ = validate.PathParam
