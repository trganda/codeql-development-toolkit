package action

import "github.com/spf13/cobra"

func NewActionCommand(base string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "action",
		Short: "Custom CodeQL action management commands",
	}
	cmd.AddCommand(newInitCommand(base))
	return cmd
}

func newInitCommand(base string) *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize GitHub Actions workflow for CodeQL",
	}
	initCmd.AddCommand(newTestInitCommand(base))
	initCmd.AddCommand(newInitBundleTestCmd(base))
	return initCmd
}
