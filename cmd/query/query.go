package query

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/advanced-security/codeql-development-toolkit/internal/language"
	tmpl "github.com/advanced-security/codeql-development-toolkit/internal/template"
)

// NewCommand returns the `query` cobra command.
func NewCommand(base, automationType *string, development, useBundle *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query feature commands",
	}
	cmd.AddCommand(newInitCmd(base, development))
	cmd.AddCommand(newGenerateCmd(base, development))
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

// newGenerateCmd returns `query generate`.
func newGenerateCmd(base *string, development *bool) *cobra.Command {
	gen := &cobra.Command{
		Use:   "generate",
		Short: "Generate CodeQL query scaffolding",
	}
	gen.AddCommand(newNewQueryCmd(base, development))
	return gen
}

// newNewQueryCmd returns `query generate new-query`.
func newNewQueryCmd(base *string, development *bool) *cobra.Command {
	var (
		queryName         string
		lang              string
		pack              string
		scope             string
		queryKind         string
		createQueryPack   bool
		createTests       bool
		overwriteExisting bool
	)

	cmd := &cobra.Command{
		Use:   "new-query",
		Short: "Create a new CodeQL query with scaffolding",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing query generate new-query command",
				"name", queryName, "language", lang, "pack", pack, "kind", queryKind)
			return runNewQuery(*base, queryName, lang, pack, scope, queryKind, createQueryPack, createTests, overwriteExisting)
		},
	}

	cmd.Flags().StringVar(&queryName, "query-name", "", "Name of the query")
	cmd.Flags().StringVar(&lang, "language", "", "Language (c|cpp|csharp|go|java|javascript|python|ruby)")
	cmd.Flags().StringVar(&pack, "pack", "", "CodeQL pack name")
	cmd.Flags().StringVar(&scope, "scope", "", "CodeQL pack scope (optional)")
	cmd.Flags().StringVar(&queryKind, "query-kind", "problem", "Query kind (problem|path-problem)")
	cmd.Flags().BoolVar(&createQueryPack, "create-query-pack", true, "Create query pack definition")
	cmd.Flags().BoolVar(&createTests, "create-tests", true, "Create test scaffolding")
	cmd.Flags().BoolVar(&overwriteExisting, "overwrite-existing", false, "Overwrite existing files")
	_ = cmd.MarkFlagRequired("query-name")
	_ = cmd.MarkFlagRequired("language")
	_ = cmd.MarkFlagRequired("pack")
	return cmd
}

// newRunCmd returns `query run`.
func newRunCmd(base *string, useBundle *bool) *cobra.Command {
	run := &cobra.Command{
		Use:   "run",
		Short: "Run query-related commands",
	}
	run.AddCommand(newInstallPacksCmd(base, useBundle))
	return run
}

// newInstallPacksCmd returns `query run install-packs`.
func newInstallPacksCmd(base *string, useBundle *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "install-packs",
		Short: "Install CodeQL packs in the repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing query run install-packs command", "base", *base, "use-bundle", *useBundle)
			return runInstallPacks(*base, *useBundle)
		},
	}
}

// queryData holds template variables for query scaffolding.
type queryData struct {
	Language         string
	QueryPackName    string
	QueryName        string
	Description      string
	QlLanguageImport string
	QueryPackFullName string
	QlLanguage       string
	QueryPackDependency string
	QueryKind        string
	TestFilePrefix   string
}

func runNewQuery(base, queryName, lang, pack, scope, queryKind string, createQueryPack, createTests, overwrite bool) error {
	langDir := language.ToDirectory(lang)
	langImport := language.ToImport(lang)
	langExt := language.ToExtension(lang)

	packFullName := pack
	if scope != "" {
		packFullName = scope + "/" + pack
	}

	// Query file goes in <base>/<langDir>/<pack>/<queryName>/<queryName>.ql
	queryDir := filepath.Join(base, langDir, pack, queryName)
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
		qlpackPath := filepath.Join(base, langDir, pack, "qlpack.yml")
		if err := tmpl.WriteFile(qlpackTmpl, qlpackPath, data, overwrite); err != nil {
			return fmt.Errorf("write qlpack-query: %w", err)
		}
	}

	// Write test scaffolding
	if createTests {
		testDir := filepath.Join(base, langDir, pack, queryName+"_tests", queryName)
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
		testPackPath := filepath.Join(base, langDir, pack, queryName+"_tests", "qlpack.yml")
		if err := tmpl.WriteFile(testPackTmpl, testPackPath, testData, overwrite); err != nil {
			return fmt.Errorf("write qlpack-test: %w", err)
		}
	}

	slog.Info("Created new query", "name", queryName, "language", lang, "pack", pack)
	return nil
}

func runInstallPacks(base string, useBundle bool) error {
	bundleFlag := ""
	if useBundle {
		bundleFlag = " --use-bundle"
	}
	baseFlag := ""
	if base != "." {
		baseFlag = fmt.Sprintf(" --base %s", base)
	}
	fmt.Printf("qlt query run install-packs%s%s\n", bundleFlag, baseFlag)
	fmt.Println("(install-packs delegates to CodeQL CLI — run codeql pack install)")
	return nil
}
