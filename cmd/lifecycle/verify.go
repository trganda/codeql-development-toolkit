package lifecycle

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVerifyCmd(_ *string) *cobra.Command {
	var lang string
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify CodeQL query quality (placeholder)",
		Long: `Verify lifecycle phase: run quality checks on compiled queries.

This phase is a placeholder. Full implementation will validate CodeQL query
metadata and run integration checks.

For now, use: qlt validation run check-queries --language <lang>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("verify: not yet fully implemented.")
			fmt.Println("Run 'qlt validation run check-queries --language <lang>' for available checks.")
			return nil
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Language to verify (e.g. go, java)")
	return cmd
}
