package lifecycle

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/query"
	qlttest "github.com/trganda/codeql-development-toolkit/internal/test"
	"github.com/trganda/codeql-development-toolkit/internal/utils"
)

func newVerifyCmd(base *string) *cobra.Command {
	var lang string
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify CodeQL query quality (placeholder)",
		Long: `Verify lifecycle phase: run quality checks on compiled queries.

Runs the full chain: install → compile → test → verify.
Requires workspace initialization (run 'qlt lifecycle init' first).

This phase is a placeholder. Full implementation will validate CodeQL query
metadata and run integration checks.

For now, use: qlt validation run check-queries --language <lang>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing lifecycle verify", "base", *base, "language", lang)
			if err := utils.CheckWorkspace(*base); err != nil {
				return err
			}
			if err := query.RunPackInstall(*base, lang); err != nil {
				return err
			}
			if err := query.RunCompile(*base, lang, "", 0); err != nil {
				return err
			}
			if err := qlttest.RunUnitTests(*base, lang, "", 4); err != nil {
				return err
			}
			fmt.Println("verify: not yet fully implemented.")
			fmt.Println("Run 'qlt validation run check-queries --language <lang>' for available checks.")
			return nil
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Language to verify (e.g. go, java)")
	return cmd
}
