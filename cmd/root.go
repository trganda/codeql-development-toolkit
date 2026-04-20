package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/cmd/action"
	"github.com/trganda/codeql-development-toolkit/cmd/bundle"
	"github.com/trganda/codeql-development-toolkit/cmd/codeql"
	"github.com/trganda/codeql-development-toolkit/cmd/pack"
	"github.com/trganda/codeql-development-toolkit/cmd/phase"
	"github.com/trganda/codeql-development-toolkit/cmd/query"
	"github.com/trganda/codeql-development-toolkit/cmd/test"
	qltlog "github.com/trganda/codeql-development-toolkit/internal/log"
)

// Global flags shared by all commands.
var (
	BasePath string
	Verbose  bool
)

var rootCmd = &cobra.Command{
	Use:   "qlt",
	Short: "CodeQL Development Lifecycle Toolkit",
	Long:  "QLT helps you develop, test, and validate CodeQL queries.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Suppress usage for logic errors from RunE. Flag/argument validation
		// errors happen before this hook runs, so they still show usage.
		cmd.SilenceUsage = true
		qltlog.Init(Verbose)
		abs, err := filepath.Abs(BasePath)
		if err != nil {
			return fmt.Errorf("resolve base path: %w", err)
		}
		BasePath = abs
		slog.Debug("QLT startup", "verbose", Verbose, "base", BasePath)
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
	rootCmd.PersistentFlags().BoolVar(&Verbose, "verbose", false, "Enable verbose logging")

	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(action.NewActionCommand(&BasePath))
	rootCmd.AddCommand(query.NewCommand(&BasePath))
	rootCmd.AddCommand(codeql.NewCommand(&BasePath))
	rootCmd.AddCommand(test.NewCommand(&BasePath))
	rootCmd.AddCommand(pack.NewCommand(&BasePath))
	rootCmd.AddCommand(bundle.NewCommand(&BasePath))
	rootCmd.AddCommand(phase.NewCommand(&BasePath))
}
