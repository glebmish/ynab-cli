package cmd

import (
	"fmt"

	"github.com/glebmish/ynab-cli/internal/validate"
	"github.com/spf13/cobra"
)

var scheduledTransactionsCmd = &cobra.Command{
	Use:   "scheduled-transactions",
	Short: "Scheduled transactions",
}

func init() {
	scheduledTransactionsCmd.PersistentFlags().Bool("flatten-splits", false, "Emit one record per subtransaction with parent fields inherited")
	scheduledTransactionsCmd.AddCommand(
		newScheduledTransactionsListCmd(),
		newScheduledTransactionsCreateCmd(),
		newScheduledTransactionsGetCmd(),
		newScheduledTransactionsUpdateCmd(),
		newScheduledTransactionsDeleteCmd(),
	)
	rootCmd.AddCommand(scheduledTransactionsCmd)
}

func newScheduledTransactionsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List scheduled transactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]string{}
			if v, _ := cmd.Flags().GetInt64("last-knowledge-of-server"); v > 0 {
				params["last_knowledge_of_server"] = fmt.Sprintf("%d", v)
			}
			return doGet(cmd, "/plans/{plan_id}/scheduled_transactions", params)
		},
	}
	cmd.Flags().Int64("last-knowledge-of-server", 0, "Server knowledge delta")
	return cmd
}

func newScheduledTransactionsCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a scheduled transaction",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonBody, err := requireJSON(cmd)
			if err != nil {
				return err
			}
			return doMutate(cmd, "POST", "/plans/{plan_id}/scheduled_transactions", nil, jsonBody)
		},
	}
	return cmd
}

func newScheduledTransactionsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a scheduled transaction by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "scheduled-transaction-id")
			if err != nil {
				return err
			}
			if err := validate.PathParam("scheduled-transaction-id", id); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/scheduled_transactions/{scheduled_transaction_id}",
				map[string]string{"scheduled_transaction_id": id})
		},
	}
	cmd.Flags().String("scheduled-transaction-id", "", "Scheduled transaction ID (required)")
	return cmd
}

func newScheduledTransactionsUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a scheduled transaction",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "scheduled-transaction-id")
			if err != nil {
				return err
			}
			jsonBody, err := requireJSON(cmd)
			if err != nil {
				return err
			}
			if err := validate.PathParam("scheduled-transaction-id", id); err != nil {
				return err
			}
			return doMutate(cmd, "PUT", "/plans/{plan_id}/scheduled_transactions/{scheduled_transaction_id}",
				map[string]string{"scheduled_transaction_id": id}, jsonBody)
		},
	}
	cmd.Flags().String("scheduled-transaction-id", "", "Scheduled transaction ID (required)")
	return cmd
}

func newScheduledTransactionsDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a scheduled transaction",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "scheduled-transaction-id")
			if err != nil {
				return err
			}
			if err := validate.PathParam("scheduled-transaction-id", id); err != nil {
				return err
			}
			return doDelete(cmd, "/plans/{plan_id}/scheduled_transactions/{scheduled_transaction_id}",
				map[string]string{"scheduled_transaction_id": id},
				"scheduled transaction", id)
		},
	}
	cmd.Flags().String("scheduled-transaction-id", "", "Scheduled transaction ID (required)")
	return cmd
}
