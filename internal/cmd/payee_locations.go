package cmd

import (
	"github.com/glebmish/ynab-cli/internal/validate"
	"github.com/spf13/cobra"
)

var payeeLocationsCmd = &cobra.Command{
	Use:   "payee-locations",
	Short: "Payee locations",
}

func init() {
	payeeLocationsCmd.AddCommand(
		newPayeeLocationsListCmd(),
		newPayeeLocationsGetCmd(),
		newPayeeLocationsListByPayeeCmd(),
	)
	rootCmd.AddCommand(payeeLocationsCmd)
}

func newPayeeLocationsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all payee locations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return doGet(cmd, "/plans/{plan_id}/payee_locations", nil)
		},
	}
}

func newPayeeLocationsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a payee location by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "payee-location-id")
			if err != nil {
				return err
			}
			if err := validate.PathParam("payee-location-id", id); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/payee_locations/{payee_location_id}",
				map[string]string{"payee_location_id": id})
		},
	}
	cmd.Flags().String("payee-location-id", "", "Payee location ID (required)")
	return cmd
}

func newPayeeLocationsListByPayeeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-by-payee",
		Short: "List payee locations for a specific payee",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requireString(cmd, "payee-id")
			if err != nil {
				return err
			}
			if err := validate.PathParam("payee-id", id); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/payees/{payee_id}/payee_locations",
				map[string]string{"payee_id": id})
		},
	}
	cmd.Flags().String("payee-id", "", "Payee ID (required)")
	return cmd
}
