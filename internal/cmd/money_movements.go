package cmd

import (
	"github.com/spf13/cobra"
)

var moneyMovementsCmd = &cobra.Command{
	Use:   "money-movements",
	Short: "Money movements and movement groups",
}

func init() {
	moneyMovementsCmd.AddCommand(
		newMoneyMovementsListCmd(),
		newMoneyMovementsListByMonthCmd(),
		newMoneyMovementsListGroupsCmd(),
		newMoneyMovementsListGroupsByMonthCmd(),
	)
	rootCmd.AddCommand(moneyMovementsCmd)
}

func newMoneyMovementsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List money movements",
		RunE: func(cmd *cobra.Command, args []string) error {
			return doGet(cmd, "/plans/{plan_id}/money_movements", nil)
		},
	}
}

func newMoneyMovementsListByMonthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-by-month",
		Short: "List money movements for a specific month",
		RunE: func(cmd *cobra.Command, args []string) error {
			month, err := requireString(cmd, "month")
			if err != nil {
				return err
			}
			if err := validateMonth(month); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/months/{month}/money_movements", map[string]string{"month": month})
		},
	}
	cmd.Flags().String("month", "", "Month (YYYY-MM-DD or 'current', required)")
	return cmd
}

func newMoneyMovementsListGroupsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-groups",
		Short: "List money movement groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			return doGet(cmd, "/plans/{plan_id}/money_movement_groups", nil)
		},
	}
}

func newMoneyMovementsListGroupsByMonthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-groups-by-month",
		Short: "List money movement groups for a specific month",
		RunE: func(cmd *cobra.Command, args []string) error {
			month, err := requireString(cmd, "month")
			if err != nil {
				return err
			}
			if err := validateMonth(month); err != nil {
				return err
			}
			return doGet(cmd, "/plans/{plan_id}/months/{month}/money_movement_groups", map[string]string{"month": month})
		},
	}
	cmd.Flags().String("month", "", "Month (YYYY-MM-DD or 'current', required)")
	return cmd
}
