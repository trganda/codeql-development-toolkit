package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/cmd/bundle"
	"github.com/trganda/codeql-development-toolkit/cmd/codeql"
	"github.com/trganda/codeql-development-toolkit/cmd/pack"
	"github.com/trganda/codeql-development-toolkit/cmd/phase"
	"github.com/trganda/codeql-development-toolkit/cmd/query"
	"github.com/trganda/codeql-development-toolkit/cmd/test"
	"github.com/trganda/codeql-development-toolkit/cmd/validation"
	qltlog "github.com/trganda/codeql-development-toolkit/internal/log"
)

// Global flags shared by all commands.
var (
	BasePath       string
	AutomationType string
	Verbose        bool
)

var rootCmd = &cobra.Command{
	Use:   "qlt",
	Short: "CodeQL Development Lifecycle Toolkit",
	Long:  "QLT helps you develop, test, and validate CodeQL queries.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		qltlog.Init(Verbose)
		slog.Debug("QLT startup", "verbose", Verbose, "base", BasePath, "automation-type", AutomationType)
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&BasePath, "base", ".", "Base repository path")
	rootCmd.PersistentFlags().StringVar(&AutomationType, "automation-type", "actions", "Automation type (e.g. actions)")
	rootCmd.PersistentFlags().BoolVar(&Verbose, "verbose", false, "Enable verbose logging")

	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(query.NewCommand(&BasePath, &AutomationType))
	rootCmd.AddCommand(codeql.NewCommand(&BasePath, &AutomationType))
	rootCmd.AddCommand(test.NewCommand(&BasePath, &AutomationType))
	rootCmd.AddCommand(validation.NewCommand(&BasePath, &AutomationType))
	rootCmd.AddCommand(pack.NewCommand(&BasePath, &AutomationType))
	rootCmd.AddCommand(bundle.NewCommand(&BasePath, &AutomationType))
	rootCmd.AddCommand(phase.NewCommand(&BasePath, &AutomationType))
}
