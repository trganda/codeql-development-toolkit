package phase

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/query"
)

func newInitCmd(base *string) *cobra.Command {
	var (
		// scope         string
		overwrite     bool
		codeqlVersion string
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the CodeQL development workspace",
		Long: `Initialize phase: set up the CodeQL development environment.

Writes codeql-workspace.yml under <base> and updates qlt.conf.json with the
provided scope and CodeQL CLI version.

Corresponds to: qlt query init`,
		Run: func(cmd *cobra.Command, args []string) {
			slog.Debug("Executing phase init", "base", *base)
			if _, err := query.InitWorkspace(*base, codeqlVersion, overwrite); err != nil {
				slog.Error("Phase init failed", "base", *base, "err", err)
				os.Exit(1)
			}
		},
	}

	// cmd.Flags().StringVar(&scope, "scope", "", "Default CodeQL pack scope (GitHub username or org, e.g. trganda)")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing files")
	cmd.Flags().StringVar(&codeqlVersion, "codeql-version", codeql.LatestCLIVersion(), "CodeQL CLI version to use (e.g. 2.25.1), auto detect latest version default;")
	return cmd
}
