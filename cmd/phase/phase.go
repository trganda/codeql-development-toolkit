package phase

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/utils"
)

// NewCommand returns the `phase` cobra command.
func NewCommand(base *string) *cobra.Command {
	common := &utils.CommonFlags{}
	cmd := &cobra.Command{
		Use:   "phase",
		Short: "CodeQL development phases",
		Long: `Execute CodeQL development phases.

The phase command mirrors Maven's build lifecycle, providing a structured,
phase-oriented workflow for CodeQL development:

  initialize  Set up the workspace and development environment
  install     Install pack dependencies
  compile     Compile CodeQL queries
  test        Run CodeQL unit tests
  verify      Validate query quality (placeholder)
  package     Create a custom CodeQL bundle
  publish     Publish CodeQL packs to the GitHub Package Registry

Phases can be run individually or in sequence. Common flows:

  qlt phase init && qlt phase install && qlt phase compile && qlt phase test

  qlt phase init && ... && qlt phase package && qlt phase publish`,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			if c.Name() == "init" {
				return
			}

			if err := utils.CheckWorkspace(*base); err != nil {
				slog.Error("Workspace check failed", "base", *base, "err", err)
				os.Exit(1)
			}
		},
	}

	cmd.PersistentFlags().StringSliceVar(&common.Packs, "pack", []string{}, "Filter by pack name (full name, can specify multiple, e.g. --pack=foo/bar --pack=baz/qux)")
	cmd.PersistentFlags().IntVar(&common.NumThreads, "num-threads", 0, "Number of threads (0 = all cores)")
	cmd.PersistentFlags().StringVar(&common.CodeQLArgs, "codeql-args", "", "Extra arguments to pass to CodeQL")

	cmd.AddCommand(newInitCmd(base))
	cmd.AddCommand(newInstallCmd(base, common))
	cmd.AddCommand(newCompileCmd(base, common))
	cmd.AddCommand(newTestCmd(base, common))
	cmd.AddCommand(newVerifyCmd(base, common))
	cmd.AddCommand(newPackageCmd(base, common))
	cmd.AddCommand(newPublishCmd(base, common))

	return cmd
}
