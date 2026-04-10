package query

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/config"
	tmpl "github.com/trganda/codeql-development-toolkit/internal/template"
)

// NewCommand returns the `query` cobra command.
func NewCommand(base, automationType *string) *cobra.Command {
	var useBundle bool
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query feature commands",
	}
	cmd.PersistentFlags().BoolVar(&useBundle, "use-bundle", false, "Use a custom CodeQL bundle")

	cmd.AddCommand(newInitCmd(base, &useBundle))
	cmd.AddCommand(newGenerateCmd(base, &useBundle))
	cmd.AddCommand(newInstallCmd(base))
	cmd.AddCommand(newCompileCmd(base))
	cmd.AddCommand(newRunCmd(base))
	return cmd
}

// newInitCmd returns `query init`.
func newInitCmd(base *string, useBundle *bool) *cobra.Command {
	var (
		overwriteExisting bool
		scope             string
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize CodeQL query workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing query init command", "base", *base, "use-bundle", *useBundle, "scope", scope)
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

			if *useBundle || scope != "" {
				cfg, err := config.LoadFromFile(*base)
				if err != nil {
					return fmt.Errorf("load config: %w", err)
				}
				if cfg == nil {
					cfg = &config.QLTConfig{}
				}
				if *useBundle {
					cfg.EnableCustomCodeQLBundles = true
				}
				if scope != "" {
					cfg.Scope = scope
				}
				if err := cfg.SaveToFile(*base); err != nil {
					return fmt.Errorf("save config: %w", err)
				}
				if *useBundle {
					slog.Info("Enabled custom CodeQL bundles in config")
				}
				if scope != "" {
					slog.Info("Saved scope to config", "scope", scope)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&overwriteExisting, "overwrite-existing", false, "Overwrite existing files")
	cmd.Flags().StringVar(&scope, "scope", "", "CodeQL pack scope (GitHub username or org, e.g. trganda)")
	return cmd
}

// newRunCmd returns `query run`.
func newRunCmd(base *string) *cobra.Command {
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
  1. Bundle binary at $HOME/.qlt/bundle/<hash>/codeql/codeql (when EnableCustomCodeQLBundles=true)
  2. CLI binary at $HOME/.qlt/packages/<hash>/codeql/codeql (installed by 'qlt codeql install')
  3. codeql found on PATH`,
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
	run.MarkFlagRequired("query")
	run.MarkFlagRequired("database")
	run.MarkFlagRequired("language")
	return run
}
