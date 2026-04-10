package test

import (
	"github.com/spf13/cobra"
)

// NewCommand returns the `test` cobra command.
func NewCommand(base, automationType *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Unit testing commands",
	}
	cmd.AddCommand(newInitCmd(base))
	cmd.AddCommand(newGetMatrixCmd(base))
	cmd.AddCommand(newRunUnitTestsCmd(base))
	cmd.AddCommand(newValidateUnitTestsCmd())
	return cmd
}
