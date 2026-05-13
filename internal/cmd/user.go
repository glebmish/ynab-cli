package cmd

import "github.com/spf13/cobra"

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Current user info",
}

func init() {
	userCmd.AddCommand(newUserGetCmd())
	rootCmd.AddCommand(userCmd)
}

func newUserGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			return doGet(cmd, "/user", nil)
		},
	}
}
