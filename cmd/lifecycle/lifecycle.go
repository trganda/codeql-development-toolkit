package lifecycle

import (
	"github.com/spf13/cobra"
)

// NewCommand returns the `lifecycle` cobra command.
func NewCommand(base, automationType *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lifecycle",
		Short: "CodeQL development lifecycle phases",
		Long: `Execute CodeQL development lifecycle phases.

The lifecycle mirrors Maven's build lifecycle, providing a structured,
phase-oriented workflow for CodeQL development:

  initialize  Set up the workspace and development environment
  install     Install pack dependencies
  compile     Compile CodeQL queries
  test        Run CodeQL unit tests
  verify      Validate query quality (placeholder)
  package     Create a custom CodeQL bundle
  publish     Publish CodeQL packs to the GitHub Package Registry

Phases can be run individually or in sequence. Common flows:

  qlt lifecycle init && qlt lifecycle install && qlt lifecycle compile && qlt lifecycle test

  qlt lifecycle init && ... && qlt lifecycle package && qlt lifecycle publish`,
	}

	cmd.AddCommand(newInitCmd(base))
	cmd.AddCommand(newInstallCmd(base))
	cmd.AddCommand(newCompileCmd(base))
	cmd.AddCommand(newTestCmd(base))
	cmd.AddCommand(newVerifyCmd(base))
	cmd.AddCommand(newPackageCmd(base))
	cmd.AddCommand(newPublishCmd(base))

	return cmd
}
