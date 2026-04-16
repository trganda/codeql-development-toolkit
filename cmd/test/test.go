package test

import (
	"github.com/spf13/cobra"
)

// NewCommand returns the `test` cobra command.
func NewCommand(base *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Unit testing commands",
	}

	cmd.AddCommand(newGetMatrixCmd(base))
	return cmd
}
