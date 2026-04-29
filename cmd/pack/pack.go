package pack

import "github.com/spf13/cobra"

// NewCommand returns the `pack` cobra command.
func NewCommand(base *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack",
		Short: "CodeQL pack management commands",
	}
	cmd.AddCommand(newListCmd(base))
	cmd.AddCommand(newResolveCmd(base))
	cmd.AddCommand(newSetCmd(base))
	cmd.AddCommand(newRunCmd(base))
	cmd.AddCommand(newGenerateCmd(base))
	cmd.AddCommand(newValidationCommand(base))
	return cmd
}
