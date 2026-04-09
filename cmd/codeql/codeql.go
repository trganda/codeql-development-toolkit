package codeql

import (
	"github.com/spf13/cobra"
)

// NewCommand returns the `codeql` cobra command.
func NewCommand(base, automationType *string, useBundle *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "codeql",
		Short: "CodeQL version management commands",
	}
	cmd.AddCommand(newSetCmd(base))
	cmd.AddCommand(newGetCmd(base))
	cmd.AddCommand(newInstallCmd(base, useBundle))
	return cmd
}
