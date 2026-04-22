package pack

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/language"

	tmpl "github.com/trganda/codeql-development-toolkit/internal/template"
)

// GeneratePackOptions holds all arguments for the pack generate new-query command.
type GeneratePackOptions struct {
	Base            string
	QueryName       string
	Lang            string
	Pack            string
	QueryKind       string
	CreateQueryPack bool
	CreateTests     bool
	Overwrite       bool
	UseBundle       bool
	Library         bool
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
	Library             bool
}

// GenerateNewPack creates a new CodeQL query with scaffolding.
// It checks whether a pack with the same full name already exists via ListPacks
// unless args.Overwrite is true; in that case it returns an error.
func GenerateNewPack(cli *codeql.CLI, opts GeneratePackOptions) error {
	langDir := language.ToDirectory(opts.Lang)
	langImport := language.ToImport(opts.Lang)
	langExt := language.ToExtension(opts.Lang)

	// Load config once — used for scope fallback and pack recording.
	cfg := config.MustLoadFromFile(opts.Base)

	packFullName := opts.Pack
	packName := GetPackName(opts.Pack)
	packScope := GetPackScope(opts.Pack)
	if packScope == "" {
		slog.Warn("No scope specificed for pack. If the pack will be shared publicly, consider adding a scope to the pack name (e.g. my-github-username/my-pack).")
	}

	// Check whether a pack with the same full name already exists.
	if !opts.Overwrite {
		targetDir := filepath.Join(opts.Base)
		existing, err := ListPacks(cli, targetDir)
		if err != nil {
			return fmt.Errorf("check existing packs: %w", err)
		}
		for _, p := range existing {
			if p.Config.FullName() == packFullName {
				return fmt.Errorf("pack %q already exists at %s; use --overwrite if you want to replace it", packFullName, p.Dir())
			}
		}
	}

	// Query file goes in <base>/<langDir>/<pack>/src/<queryName>/<queryName>.ql
	queryDir := filepath.Join(opts.Base, langDir, packName, "src", opts.QueryName)
	if err := os.MkdirAll(queryDir, 0755); err != nil {
		return fmt.Errorf("create query directory: %w", err)
	}

	// Determine template name based on kind.
	tmplName := "new-query"
	if strings.ToLower(opts.QueryKind) == "path-problem" {
		tmplName = "new-dataflow-query"
	}

	data := queryData{
		Language:            langDir,
		QueryPackName:       packName,
		QueryName:           opts.QueryName,
		Description:         "Replace this text with a description of your query.",
		QlLanguageImport:    langImport,
		QueryPackFullName:   packFullName,
		QlLanguage:          langDir,
		QueryPackDependency: packFullName,
		QueryKind:           opts.QueryKind,
		TestFilePrefix:      opts.QueryName,
		Library:             opts.Library,
	}

	slog.Debug("Creating new query", "language", opts.Lang, "dir", langDir, "pack", opts.Pack, "kind", opts.QueryKind)

	// Write query file.
	queryTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/%s.tmpl", langDir, tmplName))
	if err != nil {
		return fmt.Errorf("load query template: %w", err)
	}
	queryFilePath := filepath.Join(queryDir, opts.QueryName+".ql")
	if err := tmpl.WriteFile(queryTmpl, queryFilePath, data, opts.Overwrite); err != nil {
		return fmt.Errorf("write query file: %w", err)
	}

	// Write query pack definition.

	qlpackTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/qlpack-query.tmpl", langDir))
	if err != nil {
		return fmt.Errorf("load qlpack-query template: %w", err)
	}
	qlpackPath := filepath.Join(opts.Base, langDir, opts.Pack, "src", "qlpack.yml")
	if err := tmpl.WriteFile(qlpackTmpl, qlpackPath, data, opts.Overwrite); err != nil {
		return fmt.Errorf("write qlpack-query: %w", err)
	}

	// Write test scaffolding.
	if opts.CreateTests {
		testDir := filepath.Join(opts.Base, langDir, opts.Pack, "test", opts.QueryName)
		if err := os.MkdirAll(testDir, 0755); err != nil {
			return fmt.Errorf("create test directory: %w", err)
		}

		testTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/test.tmpl", langDir))
		if err != nil {
			return fmt.Errorf("load test template: %w", err)
		}
		testFilePath := filepath.Join(testDir, opts.QueryName+"."+langExt)
		if err := tmpl.WriteFile(testTmpl, testFilePath, data, opts.Overwrite); err != nil {
			return fmt.Errorf("write test file: %w", err)
		}

		expectedTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/expected.tmpl", langDir))
		if err != nil {
			return fmt.Errorf("load expected template: %w", err)
		}
		expectedPath := filepath.Join(testDir, opts.QueryName+".expected")
		if err := tmpl.WriteFile(expectedTmpl, expectedPath, data, opts.Overwrite); err != nil {
			return fmt.Errorf("write expected file: %w", err)
		}

		testrefTmpl, err := tmpl.Get("query/all/testref.tmpl")
		if err != nil {
			return fmt.Errorf("load testref template: %w", err)
		}
		qlrefPath := filepath.Join(testDir, opts.QueryName+".qlref")
		if err := tmpl.WriteFile(testrefTmpl, qlrefPath, data, opts.Overwrite); err != nil {
			return fmt.Errorf("write qlref file: %w", err)
		}

		testPackName := opts.Pack + "-tests"
		testPackFullName := testPackName

		testData := queryData{
			Language:            langDir,
			QueryPackName:       testPackName,
			QueryName:           opts.QueryName,
			QlLanguage:          langDir,
			QueryPackFullName:   testPackFullName,
			QueryPackDependency: packFullName,
			QueryKind:           opts.QueryKind,
			TestFilePrefix:      opts.QueryName,
		}
		testPackTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/qlpack-test.tmpl", langDir))
		if err != nil {
			return fmt.Errorf("load qlpack-test template: %w", err)
		}
		testPackPath := filepath.Join(opts.Base, langDir, opts.Pack, "test", "qlpack.yml")
		if err := tmpl.WriteFile(testPackTmpl, testPackPath, testData, opts.Overwrite); err != nil {
			return fmt.Errorf("write qlpack-test: %w", err)
		}
	}

	slog.Info("Created new query", "name", opts.QueryName, "language", opts.Lang, "pack", opts.Pack)

	// Always record the pack in config; Bundle=true only when --use-bundle was set.
	cfg.UpsertPackConfig(packFullName, opts.UseBundle)
	if err := cfg.SaveToFile(opts.Base); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	slog.Info("Recorded pack in config", "name", packFullName, "bundle", opts.UseBundle)

	return nil
}
