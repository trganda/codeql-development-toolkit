package test

import (
	"github.com/spf13/cobra"
)

// NewCommand returns the `test` cobra command.
func NewCommand(base, automationType *string, development, useBundle *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Unit testing commands",
	}
	cmd.AddCommand(newInitCmd(base, development))
	cmd.AddCommand(newRunCmd(base, development, useBundle))
	return cmd
}

// newRunCmd returns `test run`, a parent for the individual run subcommands.
func newRunCmd(base *string, development *bool, useBundle *bool) *cobra.Command {
	run := &cobra.Command{
		Use:   "run",
		Short: "Run test-related commands",
	}
	run.AddCommand(newGetMatrixCmd(base))
	run.AddCommand(newExecuteUnitTestsCmd(base, useBundle))
	run.AddCommand(newValidateUnitTestsCmd())
	return run
}
