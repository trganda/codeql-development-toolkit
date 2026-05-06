package bundle

import (
	"github.com/spf13/cobra"
)

// NewCommand returns the `bundle` cobra command.
func NewCommand(base *string) *cobra.Command {
	bundleCmd := &cobra.Command{
		Use:   "bundle",
		Short: "Custom CodeQL bundle management commands",
	}

	bundleCmd.AddCommand(newValidateIntegrationTestsCmd(base))
	return bundleCmd
}
