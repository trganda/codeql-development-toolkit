package phase

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/language"
	"github.com/trganda/codeql-development-toolkit/internal/utils"
)

// commonFlags holds flags shared across most phase subcommands.
// Populated by persistent flags on the parent and read by each subcommand.
type commonFlags struct {
	language   string
	numThreads int
	codeqlArgs string
}

// NewCommand returns the `phase` cobra command.
func NewCommand(base *string) *cobra.Command {
	common := &commonFlags{}
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
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			if c.Name() == "init" {
				return nil
			}
			if common.language != "" && common.language != "all" && !language.IsSupported(common.language) {
				return fmt.Errorf("--language must be one of %v or \"all\", got %q", language.SupportedLanguages, common.language)
			}
			return utils.CheckWorkspace(*base)
		},
	}

	cmd.PersistentFlags().StringVar(&common.language, "language", "", "Filter by language (e.g. go, java)")
	cmd.PersistentFlags().IntVar(&common.numThreads, "num-threads", 0, "Number of threads (0 = all cores)")
	cmd.PersistentFlags().StringVar(&common.codeqlArgs, "codeql-args", "", "Extra arguments to pass to CodeQL")

	cmd.AddCommand(newInitCmd(base))
	cmd.AddCommand(newInstallCmd(base, common))
	cmd.AddCommand(newCompileCmd(base, common))
	cmd.AddCommand(newTestCmd(base, common))
	cmd.AddCommand(newVerifyCmd(base, common))
	cmd.AddCommand(newPackageCmd(base, common))
	cmd.AddCommand(newPublishCmd(base, common))

	return cmd
}
