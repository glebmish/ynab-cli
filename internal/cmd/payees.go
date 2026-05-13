package cmd

import (
	"fmt"

	"github.com/glebmish/ynab-cli/internal/validate"
	"github.com/spf13/cobra"
)

var payeesCmd = &cobra.Command{
	Use:   "payees",
	Short: "Payees — list, get, create, update",
}

func init() {
	payeesCmd.AddCommand(
		newPayeesListCmd(),
		newPayeesCreateCmd(),
		newPayeesGetCmd(),
		newPayeesUpdateCmd(),
	)
	rootCmd.AddCommand(payeesCmd)
}

func newPayeesListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List payees",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]string{}
			if v, _ := cmd.Flags().GetInt64("last-knowledge-of-server"); v > 0 {
				params["last_knowledge_of_server"] = fmt.Sprintf("%d", v)
			}
			return doGet(cmd, "/plans/{plan_id}/payees", params)
		},
	}
	cmd.Flags().Int64("last-knowledge-of-server", 0, "Server knowledge delta")
	return cmd
}

func newPayeesCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a payee",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonBody, err := requireJSON(cmd)
			if err != nil {
				return err
			}
			return doMutate(cmd, "POST", "/plans/{plan_id}/payees", nil, jsonBody)
		},
	}
	return cmd
}

func newPayeesGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a payee by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "payee-id")
			if err != nil {
				return err
			}
			if err := validate.PathParam("payee-id", id); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/payees/{payee_id}", map[string]string{"payee_id": id})
		},
	}
	cmd.Flags().String("payee-id", "", "Payee ID (required)")
	return cmd
}

func newPayeesUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a payee",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "payee-id")
			if err != nil {
				return err
			}
			jsonBody, err := requireJSON(cmd)
			if err != nil {
				return err
			}
			if err := validate.PathParam("payee-id", id); err != nil {
				return err
			}
			return doMutate(cmd, "PATCH", "/plans/{plan_id}/payees/{payee_id}",
				map[string]string{"payee_id": id}, jsonBody)
		},
	}
	cmd.Flags().String("payee-id", "", "Payee ID (required)")
	return cmd
}
