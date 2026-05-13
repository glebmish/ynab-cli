package cmd

import (
	"fmt"

	"github.com/glebmish/ynab-cli/internal/validate"
	"github.com/spf13/cobra"
)

var transactionsCmd = &cobra.Command{
	Use:   "transactions",
	Short: "Transactions — list, get, create, update, import, delete",
}

func init() {
	transactionsCmd.PersistentFlags().Bool("flatten-splits", false, "Emit one record per subtransaction with parent fields inherited")
	transactionsCmd.AddCommand(
		newTransactionsListCmd(),
		newTransactionsCreateCmd(),
		newTransactionsUpdateBulkCmd(),
		newTransactionsImportCmd(),
		newTransactionsGetCmd(),
		newTransactionsUpdateCmd(),
		newTransactionsDeleteCmd(),
		newTransactionsListByAccountCmd(),
		newTransactionsListByCategoryCmd(),
		newTransactionsListByPayeeCmd(),
		newTransactionsListByMonthCmd(),
	)
	rootCmd.AddCommand(transactionsCmd)
}

func addTransactionListFlags(cmd *cobra.Command) {
	cmd.Flags().String("since-date", "", "Only transactions on or after this date (YYYY-MM-DD)")
	cmd.Flags().String("type", "", "Transaction type filter (uncategorized|unapproved)")
	cmd.Flags().Int64("last-knowledge-of-server", 0, "Server knowledge delta")
}

func applyTransactionListFlags(cmd *cobra.Command, params map[string]string) error {
	if v, _ := cmd.Flags().GetString("since-date"); v != "" {
		if err := validate.DateParam("since-date", v); err != nil {
			return err
		}
		params["since_date"] = v
	}
	if v, _ := cmd.Flags().GetString("type"); v != "" {
		params["type"] = v
	}
	if v, _ := cmd.Flags().GetInt64("last-knowledge-of-server"); v > 0 {
		params["last_knowledge_of_server"] = fmt.Sprintf("%d", v)
	}
	return nil
}

func newTransactionsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List transactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]string{}
			if err := applyTransactionListFlags(cmd, params); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/transactions", params)
		},
	}
	addTransactionListFlags(cmd)
	return cmd
}

func newTransactionsCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a transaction (single or bulk via transactions array)",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonBody, err := requireJSON(cmd)
			if err != nil {
				return err
			}
			return doMutate(cmd, "POST", "/plans/{plan_id}/transactions", nil, jsonBody)
		},
	}
	return cmd
}

func newTransactionsUpdateBulkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-bulk",
		Short: "Update multiple transactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonBody, err := requireJSON(cmd)
			if err != nil {
				return err
			}
			return doMutate(cmd, "PATCH", "/plans/{plan_id}/transactions", nil, jsonBody)
		},
	}
	return cmd
}

func newTransactionsImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import",
		Short: "Import transactions from linked accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return doMutate(cmd, "POST", "/plans/{plan_id}/transactions/import", nil, "")
		},
	}
}

func newTransactionsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a transaction by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "transaction-id")
			if err != nil {
				return err
			}
			if err := validate.PathParam("transaction-id", id); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/transactions/{transaction_id}",
				map[string]string{"transaction_id": id})
		},
	}
	cmd.Flags().String("transaction-id", "", "Transaction ID (required)")
	return cmd
}

func newTransactionsUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a transaction",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "transaction-id")
			if err != nil {
				return err
			}
			jsonBody, err := requireJSON(cmd)
			if err != nil {
				return err
			}
			if err := validate.PathParam("transaction-id", id); err != nil {
				return err
			}
			return doMutate(cmd, "PUT", "/plans/{plan_id}/transactions/{transaction_id}",
				map[string]string{"transaction_id": id}, jsonBody)
		},
	}
	cmd.Flags().String("transaction-id", "", "Transaction ID (required)")
	return cmd
}

func newTransactionsDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a transaction",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "transaction-id")
			if err != nil {
				return err
			}
			if err := validate.PathParam("transaction-id", id); err != nil {
				return err
			}
			return doDelete(cmd, "/plans/{plan_id}/transactions/{transaction_id}",
				map[string]string{"transaction_id": id}, "transaction", id)
		},
	}
	cmd.Flags().String("transaction-id", "", "Transaction ID (required)")
	return cmd
}

func newTransactionsListByAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-by-account",
		Short: "List transactions for an account",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "account-id")
			if err != nil {
				return err
			}
			if err := validate.PathParam("account-id", id); err != nil {
				return err
			}
			params := map[string]string{"account_id": id}
			if err := applyTransactionListFlags(cmd, params); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/accounts/{account_id}/transactions", params)
		},
	}
	cmd.Flags().String("account-id", "", "Account ID (required)")
	addTransactionListFlags(cmd)
	return cmd
}

func newTransactionsListByCategoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-by-category",
		Short: "List transactions for a category",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "category-id")
			if err != nil {
				return err
			}
			if err := validate.PathParam("category-id", id); err != nil {
				return err
			}
			params := map[string]string{"category_id": id}
			if err := applyTransactionListFlags(cmd, params); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/categories/{category_id}/transactions", params)
		},
	}
	cmd.Flags().String("category-id", "", "Category ID (required)")
	addTransactionListFlags(cmd)
	return cmd
}

func newTransactionsListByPayeeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-by-payee",
		Short: "List transactions for a payee",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "payee-id")
			if err != nil {
				return err
			}
			if err := validate.PathParam("payee-id", id); err != nil {
				return err
			}
			params := map[string]string{"payee_id": id}
			if err := applyTransactionListFlags(cmd, params); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/payees/{payee_id}/transactions", params)
		},
	}
	cmd.Flags().String("payee-id", "", "Payee ID (required)")
	addTransactionListFlags(cmd)
	return cmd
}

func newTransactionsListByMonthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-by-month",
		Short: "List transactions for a month",
		RunE: func(cmd *cobra.Command, args []string) error {
			month, err := requireString(cmd, "month")
			if err != nil {
				return err
			}
			if err := validateMonth(month); err != nil {
				return err
			}
			params := map[string]string{"month": month}
			if err := applyTransactionListFlags(cmd, params); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/months/{month}/transactions", params)
		},
	}
	cmd.Flags().String("month", "", "Month (YYYY-MM-DD or 'current', required)")
	addTransactionListFlags(cmd)
	return cmd
}
