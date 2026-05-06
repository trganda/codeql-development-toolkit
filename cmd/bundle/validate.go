package bundle

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/bundle"
)

// newValidateIntegrationTestsCmd returns `bundle run validate-integration-tests`.
func newValidateIntegrationTestsCmd(_ *string) *cobra.Command {
	var expected, actual string
	cmd := &cobra.Command{
		Use:   "validate-integration-tests",
		Short: "Validate bundle integration test SARIF results",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing bundle run validate-integration-tests command", "expected", expected, "actual", actual)
			slog.Info("Comparing SARIF results", "expected", expected, "actual", actual)
			return bundle.Validate(&bundle.ValidateOptions{Expected: expected, Actual: actual})
		},
	}
	cmd.Flags().StringVar(&expected, "expected", "", "Path to expected SARIF file")
	cmd.Flags().StringVar(&actual, "actual", "", "Path to actual SARIF file")
	_ = cmd.MarkFlagRequired("expected")
	_ = cmd.MarkFlagRequired("actual")
	return cmd
}
