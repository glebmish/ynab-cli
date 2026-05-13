package cmd

import (
	"fmt"

	"github.com/glebmish/ynab-cli/internal/validate"
	"github.com/spf13/cobra"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Accounts — list, get, create",
}

func init() {
	accountsCmd.AddCommand(
		newAccountsListCmd(),
		newAccountsGetCmd(),
		newAccountsCreateCmd(),
	)
	rootCmd.AddCommand(accountsCmd)
}

func newAccountsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List accounts in a plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]string{}
			if v, _ := cmd.Flags().GetInt64("last-knowledge-of-server"); v > 0 {
				params["last_knowledge_of_server"] = fmt.Sprintf("%d", v)
			}
			return doGet(cmd, "/plans/{plan_id}/accounts", params)
		},
	}
	cmd.Flags().Int64("last-knowledge-of-server", 0, "Server knowledge delta")
	return cmd
}

func newAccountsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get an account by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			accountID, err := requireString(cmd, "account-id")
			if err != nil {
				return err
			}
			if err := validate.PathParam("account-id", accountID); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/accounts/{account_id}", map[string]string{"account_id": accountID})
		},
	}
	cmd.Flags().String("account-id", "", "Account ID (required)")
	return cmd
}

func newAccountsCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an account",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonBody, err := requireJSON(cmd)
			if err != nil {
				return err
			}
			return doMutate(cmd, "POST", "/plans/{plan_id}/accounts", nil, jsonBody)
		},
	}
	return cmd
}
