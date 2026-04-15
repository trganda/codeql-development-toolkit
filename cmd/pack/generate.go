package pack

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	tmpl "github.com/trganda/codeql-development-toolkit/internal/template"
)

// newGenerateCmd returns `pack generate`.
func newGenerateCmd(base string) *cobra.Command {
	var bundle bool
	gen := &cobra.Command{
		Use:   "generate",
		Short: "Generate CodeQL query scaffolding",
	}
	gen.AddCommand(newNewQueryCmd(base, bundle))

	gen.Flags().BoolVar(&bundle, "bundle", false, "Add to a custom CodeQL bundle")
	return gen
}

// newNewQueryCmd returns `pack generate new-query`.
func newNewQueryCmd(base string, useBundle bool) *cobra.Command {
	var (
		queryName       string
		lang            string
		pack            string
		scope           string
		queryKind       string
		createQueryPack bool
		createTests     bool
		overwrite       bool
	)

	cmd := &cobra.Command{
		Use:   "new-query",
		Short: "Create a new CodeQL query with scaffolding",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing query generate new-query command",
				"name", queryName, "language", lang, "pack", pack, "kind", queryKind, "use-bundle", useBundle)
			return runNewQuery(base, queryName, lang, pack, scope, queryKind, createQueryPack, createTests, overwrite, useBundle)
		},
	}

	cmd.Flags().StringVar(&queryName, "query-name", "", "Name of the query")
	cmd.Flags().StringVar(&lang, "language", "", "Language (c|cpp|csharp|go|java|javascript|python|ruby)")
	cmd.Flags().StringVar(&pack, "pack", "", "CodeQL pack name")
	cmd.Flags().StringVar(&scope, "scope", "", "CodeQL pack scope (optional), use globally configured scope in qlt.conf.json if not provided")
	cmd.Flags().StringVar(&queryKind, "query-kind", "problem", "Query kind (problem|path-problem)")
	cmd.Flags().BoolVar(&createQueryPack, "create-query-pack", true, "Create query pack definition")
	cmd.Flags().BoolVar(&createTests, "create-tests", true, "Create test scaffolding")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing files")
	cmd.MarkFlagRequired("query-name")
	cmd.MarkFlagRequired("language")
	cmd.MarkFlagRequired("pack")
	return cmd
}

// queryData holds template variables for query scaffolding.
type queryData struct {
	Language            string
	QueryPackName       string
	QueryName           string
	Description         string
	QlLanguageImport    string
	QueryPackFullName   string
	QlLanguage          string
	QueryPackDependency string
	QueryKind           string
	TestFilePrefix      string
}

func runNewQuery(base, queryName, lang, pack, scope, queryKind string, createQueryPack, createTests, overwrite bool, useBundle bool) error {
	langDir := language.ToDirectory(lang)
	langImport := language.ToImport(lang)
	langExt := language.ToExtension(lang)

	// Load config once — used for scope fallback and pack recording.
	cfg, err := config.LoadFromFile(base)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg == nil {
		cfg = &config.QLTConfig{}
	}

	// Fall back to the scope stored in config when --scope is not provided.
	if scope == "" && cfg.Scope != "" {
		scope = cfg.Scope
	}

	packFullName := pack
	if scope != "" {
		packFullName = scope + "/" + pack
	}

	// Query file goes in <base>/<langDir>/<pack>/src/<queryName>/<queryName>.ql
	queryDir := filepath.Join(base, langDir, pack, "src", queryName)
	if err := os.MkdirAll(queryDir, 0755); err != nil {
		return fmt.Errorf("create query directory: %w", err)
	}

	// Determine template name based on kind
	tmplName := "new-query"
	if strings.ToLower(queryKind) == "path-problem" {
		tmplName = "new-dataflow-query"
	}

	data := queryData{
		Language:            langDir,
		QueryPackName:       pack,
		QueryName:           queryName,
		Description:         "Replace this text with a description of your query.",
		QlLanguageImport:    langImport,
		QueryPackFullName:   packFullName,
		QlLanguage:          langDir,
		QueryPackDependency: packFullName,
		QueryKind:           queryKind,
		TestFilePrefix:      queryName,
	}

	slog.Debug("Creating new query", "language", lang, "dir", langDir, "pack", pack, "kind", queryKind)

	// Write query file
	queryTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/%s.tmpl", langDir, tmplName))
	if err != nil {
		return fmt.Errorf("load query template: %w", err)
	}
	queryFilePath := filepath.Join(queryDir, queryName+".ql")
	if err := tmpl.WriteFile(queryTmpl, queryFilePath, data, overwrite); err != nil {
		return fmt.Errorf("write query file: %w", err)
	}

	// Write query pack definition
	if createQueryPack {
		qlpackTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/qlpack-query.tmpl", langDir))
		if err != nil {
			return fmt.Errorf("load qlpack-query template: %w", err)
		}
		qlpackPath := filepath.Join(base, langDir, pack, "src", "qlpack.yml")
		if err := tmpl.WriteFile(qlpackTmpl, qlpackPath, data, overwrite); err != nil {
			return fmt.Errorf("write qlpack-query: %w", err)
		}
	}

	// Write test scaffolding
	if createTests {
		testDir := filepath.Join(base, langDir, pack, "test", queryName)
		if err := os.MkdirAll(testDir, 0755); err != nil {
			return fmt.Errorf("create test directory: %w", err)
		}

		// test source file
		testTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/test.tmpl", langDir))
		if err != nil {
			return fmt.Errorf("load test template: %w", err)
		}
		testFilePath := filepath.Join(testDir, queryName+"."+langExt)
		if err := tmpl.WriteFile(testTmpl, testFilePath, data, overwrite); err != nil {
			return fmt.Errorf("write test file: %w", err)
		}

		// expected output
		expectedTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/expected.tmpl", langDir))
		if err != nil {
			return fmt.Errorf("load expected template: %w", err)
		}
		expectedPath := filepath.Join(testDir, queryName+".expected")
		if err := tmpl.WriteFile(expectedTmpl, expectedPath, data, overwrite); err != nil {
			return fmt.Errorf("write expected file: %w", err)
		}

		// qlref file
		testrefTmpl, err := tmpl.Get("query/all/testref.tmpl")
		if err != nil {
			return fmt.Errorf("load testref template: %w", err)
		}
		qlrefPath := filepath.Join(testDir, queryName+".qlref")
		if err := tmpl.WriteFile(testrefTmpl, qlrefPath, data, overwrite); err != nil {
			return fmt.Errorf("write qlref file: %w", err)
		}

		// test qlpack
		testPackName := pack + "-tests"
		testPackFullName := testPackName
		if scope != "" {
			testPackFullName = scope + "/" + testPackName
		}
		testData := queryData{
			Language:            langDir,
			QueryPackName:       testPackName,
			QueryName:           queryName,
			QlLanguage:          langDir,
			QueryPackFullName:   testPackFullName,
			QueryPackDependency: packFullName,
			QueryKind:           queryKind,
			TestFilePrefix:      queryName,
		}
		testPackTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/qlpack-test.tmpl", langDir))
		if err != nil {
			return fmt.Errorf("load qlpack-test template: %w", err)
		}
		testPackPath := filepath.Join(base, langDir, pack, "test", "qlpack.yml")
		if err := tmpl.WriteFile(testPackTmpl, testPackPath, testData, overwrite); err != nil {
			return fmt.Errorf("write qlpack-test: %w", err)
		}
	}

	slog.Info("Created new query", "name", queryName, "language", lang, "pack", pack)

	// Always record the pack in config; Bundle=true only when --use-bundle was set.
	cfg.UpsertPackConfig(packFullName, useBundle)
	if err := cfg.SaveToFile(base); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	slog.Info("Recorded pack in config", "name", packFullName, "bundle", useBundle)

	return nil
}
