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

// GenerateArgs holds all arguments for the pack generate new-query command.
type GenerateArgs struct {
	Base            string
	QueryName       string
	Lang            string
	Pack            string
	Scope           string
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
func GenerateNewPack(cli *codeql.CLI, args GenerateArgs) error {
	langDir := language.ToDirectory(args.Lang)
	langImport := language.ToImport(args.Lang)
	langExt := language.ToExtension(args.Lang)

	// Load config once — used for scope fallback and pack recording.
	cfg := config.MustLoadFromFile(args.Base)

	// Fall back to the scope stored in config when --scope is not provided.
	if args.Scope == "" && cfg.Scope != "" {
		args.Scope = cfg.Scope
	}

	packFullName := args.Pack
	if args.Scope != "" {
		packFullName = args.Scope + "/" + args.Pack
	}

	// Check whether a pack with the same full name already exists.
	if !args.Overwrite {
		targetDir := filepath.Join(args.Base)
		existing, err := ListPacks(cli, targetDir)
		if err != nil {
			return fmt.Errorf("check existing packs: %w", err)
		}
		for _, p := range existing {
			if p.Config.FullName() == packFullName {
				return fmt.Errorf("pack %q already exists at %s; use --overwrite to replace it", packFullName, p.Dir())
			}
		}
	}

	// Query file goes in <base>/<langDir>/<pack>/src/<queryName>/<queryName>.ql
	queryDir := filepath.Join(args.Base, langDir, args.Pack, "src", args.QueryName)
	if err := os.MkdirAll(queryDir, 0755); err != nil {
		return fmt.Errorf("create query directory: %w", err)
	}

	// Determine template name based on kind.
	tmplName := "new-query"
	if strings.ToLower(args.QueryKind) == "path-problem" {
		tmplName = "new-dataflow-query"
	}

	data := queryData{
		Language:            langDir,
		QueryPackName:       args.Pack,
		QueryName:           args.QueryName,
		Description:         "Replace this text with a description of your query.",
		QlLanguageImport:    langImport,
		QueryPackFullName:   packFullName,
		QlLanguage:          langDir,
		QueryPackDependency: packFullName,
		QueryKind:           args.QueryKind,
		TestFilePrefix:      args.QueryName,
		Library:             args.Library,
	}

	slog.Debug("Creating new query", "language", args.Lang, "dir", langDir, "pack", args.Pack, "kind", args.QueryKind)

	// Write query file.
	queryTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/%s.tmpl", langDir, tmplName))
	if err != nil {
		return fmt.Errorf("load query template: %w", err)
	}
	queryFilePath := filepath.Join(queryDir, args.QueryName+".ql")
	if err := tmpl.WriteFile(queryTmpl, queryFilePath, data, args.Overwrite); err != nil {
		return fmt.Errorf("write query file: %w", err)
	}

	// Write query pack definition.
	if args.CreateQueryPack {
		qlpackTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/qlpack-query.tmpl", langDir))
		if err != nil {
			return fmt.Errorf("load qlpack-query template: %w", err)
		}
		qlpackPath := filepath.Join(args.Base, langDir, args.Pack, "src", "qlpack.yml")
		if err := tmpl.WriteFile(qlpackTmpl, qlpackPath, data, args.Overwrite); err != nil {
			return fmt.Errorf("write qlpack-query: %w", err)
		}
	}

	// Write test scaffolding.
	if args.CreateTests {
		testDir := filepath.Join(args.Base, langDir, args.Pack, "test", args.QueryName)
		if err := os.MkdirAll(testDir, 0755); err != nil {
			return fmt.Errorf("create test directory: %w", err)
		}

		testTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/test.tmpl", langDir))
		if err != nil {
			return fmt.Errorf("load test template: %w", err)
		}
		testFilePath := filepath.Join(testDir, args.QueryName+"."+langExt)
		if err := tmpl.WriteFile(testTmpl, testFilePath, data, args.Overwrite); err != nil {
			return fmt.Errorf("write test file: %w", err)
		}

		expectedTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/expected.tmpl", langDir))
		if err != nil {
			return fmt.Errorf("load expected template: %w", err)
		}
		expectedPath := filepath.Join(testDir, args.QueryName+".expected")
		if err := tmpl.WriteFile(expectedTmpl, expectedPath, data, args.Overwrite); err != nil {
			return fmt.Errorf("write expected file: %w", err)
		}

		testrefTmpl, err := tmpl.Get("query/all/testref.tmpl")
		if err != nil {
			return fmt.Errorf("load testref template: %w", err)
		}
		qlrefPath := filepath.Join(testDir, args.QueryName+".qlref")
		if err := tmpl.WriteFile(testrefTmpl, qlrefPath, data, args.Overwrite); err != nil {
			return fmt.Errorf("write qlref file: %w", err)
		}

		testPackName := args.Pack + "-tests"
		testPackFullName := testPackName
		if args.Scope != "" {
			testPackFullName = args.Scope + "/" + testPackName
		}
		testData := queryData{
			Language:            langDir,
			QueryPackName:       testPackName,
			QueryName:           args.QueryName,
			QlLanguage:          langDir,
			QueryPackFullName:   testPackFullName,
			QueryPackDependency: packFullName,
			QueryKind:           args.QueryKind,
			TestFilePrefix:      args.QueryName,
		}
		testPackTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/qlpack-test.tmpl", langDir))
		if err != nil {
			return fmt.Errorf("load qlpack-test template: %w", err)
		}
		testPackPath := filepath.Join(args.Base, langDir, args.Pack, "test", "qlpack.yml")
		if err := tmpl.WriteFile(testPackTmpl, testPackPath, testData, args.Overwrite); err != nil {
			return fmt.Errorf("write qlpack-test: %w", err)
		}
	}

	slog.Info("Created new query", "name", args.QueryName, "language", args.Lang, "pack", args.Pack)

	// Always record the pack in config; Bundle=true only when --use-bundle was set.
	cfg.UpsertPackConfig(packFullName, args.UseBundle)
	if err := cfg.SaveToFile(args.Base); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	slog.Info("Recorded pack in config", "name", packFullName, "bundle", args.UseBundle)

	return nil
}
