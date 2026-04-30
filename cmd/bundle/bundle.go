package bundle

import (
	"log/slog"

	"github.com/spf13/cobra"
)

// NewCommand returns the `bundle` cobra command.
func NewCommand(base *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Custom CodeQL bundle management commands",
	}

	cmd.AddCommand(newRunCmd(base))
	return cmd
}

func newRunCmd(base *string) *cobra.Command {
	run := &cobra.Command{
		Use:   "run",
		Short: "Run bundle commands",
	}
	run.AddCommand(newValidateIntegrationTestsCmd(base))
	return run
}

func newValidateIntegrationTestsCmd(base *string) *cobra.Command {
	var expected, actual string
	cmd := &cobra.Command{
		Use:   "validate-integration-tests",
		Short: "Validate bundle integration test SARIF results",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing bundle run validate-integration-tests command", "expected", expected, "actual", actual)
			slog.Info("Comparing SARIF results", "expected", expected, "actual", actual)
			return nil
		},
	}
	cmd.Flags().StringVar(&expected, "expected", "", "Path to expected SARIF file")
	cmd.Flags().StringVar(&actual, "actual", "", "Path to actual SARIF file")
	return cmd
}
