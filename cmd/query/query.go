package query

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	tmpl "github.com/trganda/codeql-development-toolkit/internal/template"
)

// NewCommand returns the `query` cobra command.
func NewCommand(base, automationType *string, development, useBundle *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query feature commands",
	}
	cmd.AddCommand(newInitCmd(base, development))
	cmd.AddCommand(newGenerateCmd(base, development))
	cmd.AddCommand(newInstallCmd(base))
	cmd.AddCommand(newCompileCmd(base))
	cmd.AddCommand(newRunCmd(base, useBundle))
	return cmd
}

// newInitCmd returns `query init`.
func newInitCmd(base *string, development *bool) *cobra.Command {
	var overwriteExisting bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize CodeQL query workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing query init command", "base", *base)
			if err := os.MkdirAll(*base, 0755); err != nil {
				return fmt.Errorf("create base directory: %w", err)
			}
			tmplContent, err := tmpl.Get("query/codeql-workspace.tmpl")
			if err != nil {
				return fmt.Errorf("load workspace template: %w", err)
			}
			dst := filepath.Join(*base, "codeql-workspace.yml")
			if err := tmpl.WriteFile(tmplContent, dst, nil, overwriteExisting); err != nil {
				return err
			}
			slog.Info("Initialized CodeQL workspace", "path", dst)
			return nil
		},
	}
	cmd.Flags().BoolVar(&overwriteExisting, "overwrite-existing", false, "Overwrite existing files")
	return cmd
}

// newRunCmd returns `query run`.
func newRunCmd(base *string, useBundle *bool) *cobra.Command {
	var (
		queryName       string
		database        string
		lang            string
		pack            string
		format          string
		output          string
		additionalPacks string
		threads         int
	)
	run := &cobra.Command{
		Use:   "run",
		Short: "Run a CodeQL query against a database",
		Long: `Run a CodeQL query against a database using 'codeql database analyze'.

The query is located by name in order:
  1. Config lookup in qlt.conf.json (recorded by 'qlt query generate new-query')
  2. Filesystem search up to 3 levels under <base>/<language>/[pack]

The CodeQL binary is resolved in order:
  1. $HOME/.qlt/codeql/<version>/codeql/codeql (installed by 'qlt codeql install';
     version read from qlt.conf.json)
  2. codeql found on PATH`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing query run",
				"query", queryName, "database", database, "language", lang,
				"pack", pack, "format", format, "output", output, "threads", threads)
			return runQuery(*base, queryName, database, lang, pack, format, output, additionalPacks, threads)
		},
	}
	run.Flags().StringVar(&queryName, "query", "", "Query name to run, e.g. MyQuery (required)")
	run.Flags().StringVar(&database, "database", "", "Path to the CodeQL database (required)")
	run.Flags().StringVar(&lang, "language", "", "Language of the query, e.g. go, java (required)")
	run.Flags().StringVar(&pack, "pack", "", "Pack name to narrow the search (optional)")
	run.Flags().StringVar(&format, "format", "sarif-latest", "Output format: sarif-latest, csv, text, dot, bqrs")
	run.Flags().StringVar(&output, "output", "", "Output file path (default: <query-name>.<ext> beside the query file)")
	run.Flags().StringVar(&additionalPacks, "additional-packs", "", "Colon-separated list of additional pack search paths")
	run.Flags().IntVar(&threads, "threads", 0, "Number of threads (0 = use all available cores)")
	_ = run.MarkFlagRequired("query")
	_ = run.MarkFlagRequired("database")
	_ = run.MarkFlagRequired("language")
	return run
}
