package query

import (
	"github.com/spf13/cobra"
)

// NewCommand returns the `query` cobra command.
func NewCommand(base *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query feature commands",
	}

	cmd.AddCommand(newRunCmd(base))
	cmd.AddCommand(newGenerateCmd(base))

	return cmd
}
