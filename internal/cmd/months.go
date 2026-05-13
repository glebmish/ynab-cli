package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var monthsCmd = &cobra.Command{
	Use:   "months",
	Short: "Plan months",
}

func init() {
	monthsCmd.AddCommand(
		newMonthsListCmd(),
		newMonthsGetCmd(),
	)
	rootCmd.AddCommand(monthsCmd)
}

func newMonthsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List months for the plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]string{}
			if v, _ := cmd.Flags().GetInt64("last-knowledge-of-server"); v > 0 {
				params["last_knowledge_of_server"] = fmt.Sprintf("%d", v)
			}
			return doGet(cmd, "/plans/{plan_id}/months", params)
		},
	}
	cmd.Flags().Int64("last-knowledge-of-server", 0, "Server knowledge delta")
	return cmd
}

func newMonthsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a specific month",
		RunE: func(cmd *cobra.Command, args []string) error {
			month, err := requireString(cmd, "month")
			if err != nil {
				return err
			}
			if err := validateMonth(month); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/months/{month}", map[string]string{"month": month})
		},
	}
	cmd.Flags().String("month", "", "Month (YYYY-MM-DD or 'current', required)")
	return cmd
}
