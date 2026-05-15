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
	Base      string
	QueryName string
	Lang      string
	Pack      string
	QueryKind string
	SkipTest  bool
	Overwrite bool
	UseBundle bool
	Library   bool
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

// testPackSuffix is appended to the normal pack's full name to derive the test pack's full name.
const testPackSuffix = "-tests"

// GenerateNewPack creates a new CodeQL query pack, and (unless SkipTest is set) a
// matching test pack. Existence checks are scoped per pack: the normal pack must
// not already exist (unless Overwrite is set), and when SkipTest is false the
// test pack must not already exist (unless Overwrite is set).
func GenerateNewPack(cli *codeql.CLI, opts GeneratePackOptions) error {
	if err := writeNormalPack(cli, opts); err != nil {
		return err
	}
	if !opts.SkipTest {
		if err := writeTestPack(cli, opts); err != nil {
			return err
		}
	}
	return nil
}

// AddTestPack creates the test pack scaffolding for an already-existing normal
// pack. It refuses to run if the normal pack is missing, regardless of Overwrite.
// The test pack must not already exist unless Overwrite is set.
func AddTestPack(cli *codeql.CLI, opts GeneratePackOptions) error {
	langDir := language.ToDirectory(opts.Lang)
	packFullName := opts.Pack

	existing, err := ListPacks(cli, opts.Base)
	if err != nil {
		return fmt.Errorf("check existing packs: %w", err)
	}
	var found bool
	for _, p := range existing {
		if p.Config.FullName() == packFullName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("pack %q does not exist under %s/%s/; run \"qlt pack generate\" first", packFullName, opts.Base, langDir)
	}

	return writeTestPack(cli, opts)
}

// writeNormalPack creates the normal qlpack at <base>/<langDir>/<packName>/src/
// and records the pack in qlt.conf.json. Overwrite only affects files under src/.
func writeNormalPack(cli *codeql.CLI, opts GeneratePackOptions) error {
	langDir := language.ToDirectory(opts.Lang)
	langImport := language.ToImport(opts.Lang)

	cfg := config.MustLoadFromFile(opts.Base)

	packFullName := opts.Pack
	packName := GetPackName(opts.Pack)
	packScope := GetPackScope(opts.Pack)
	if packScope == "" {
		slog.Warn("No scope specificed for pack. If the pack will be shared publicly, consider adding a scope to the pack name (e.g. my-github-username/my-pack).")
	}

	if !opts.Overwrite {
		existing, err := ListPacks(cli, opts.Base)
		if err != nil {
			return fmt.Errorf("check existing packs: %w", err)
		}
		for _, p := range existing {
			if p.Config.FullName() == packFullName {
				return fmt.Errorf("pack %q already exists at %s; use --overwrite if you want to replace it", packFullName, p.Dir())
			}
		}
	}

	queryDir := filepath.Join(opts.Base, langDir, packName, "src", opts.QueryName)
	if err := os.MkdirAll(queryDir, 0700); err != nil {
		return fmt.Errorf("create query directory: %w", err)
	}

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

	queryTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/%s.tmpl", langDir, tmplName))
	if err != nil {
		return fmt.Errorf("load query template: %w", err)
	}
	queryFilePath := filepath.Join(queryDir, opts.QueryName+".ql")
	if err := tmpl.WriteFile(queryTmpl, queryFilePath, data, opts.Overwrite); err != nil {
		return fmt.Errorf("write query file: %w", err)
	}

	qlpackTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/qlpack-query.tmpl", langDir))
	if err != nil {
		return fmt.Errorf("load qlpack-query template: %w", err)
	}
	qlpackPath := filepath.Join(opts.Base, langDir, packName, "src", "qlpack.yml")
	if err := tmpl.WriteFile(qlpackTmpl, qlpackPath, data, opts.Overwrite); err != nil {
		return fmt.Errorf("write qlpack-query: %w", err)
	}

	slog.Info("Created new query", "name", opts.QueryName, "language", opts.Lang, "pack", opts.Pack)

	cfg.UpsertPackConfig(packFullName, opts.UseBundle)
	if err := cfg.SaveToFile(opts.Base); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	slog.Info("Recorded pack in config", "name", packFullName, "bundle", opts.UseBundle)

	return nil
}

// writeTestPack creates the test qlpack at <base>/<langDir>/<packName>/test/.
// Overwrite only affects files under test/. The test pack must not already
// exist unless Overwrite is set; the test pack is not recorded in qlt.conf.json.
func writeTestPack(cli *codeql.CLI, opts GeneratePackOptions) error {
	langDir := language.ToDirectory(opts.Lang)
	langExt := language.ToExtension(opts.Lang)

	packFullName := opts.Pack
	packName := GetPackName(opts.Pack)
	testPackFullName := packFullName + testPackSuffix

	if !opts.Overwrite {
		existing, err := ListPacks(cli, opts.Base)
		if err != nil {
			return fmt.Errorf("check existing test packs: %w", err)
		}
		for _, p := range existing {
			if p.Config.FullName() == testPackFullName {
				return fmt.Errorf("test pack %q already exists at %s; use --overwrite if you want to replace it", testPackFullName, p.Dir())
			}
		}
	}

	testDir := filepath.Join(opts.Base, langDir, packName, "test", opts.QueryName)
	if err := os.MkdirAll(testDir, 0700); err != nil {
		return fmt.Errorf("create test directory: %w", err)
	}

	data := queryData{
		Language:            langDir,
		QueryPackName:       packName,
		QueryName:           opts.QueryName,
		QlLanguage:          langDir,
		QueryPackFullName:   packFullName,
		QueryPackDependency: packFullName,
		TestFilePrefix:      opts.QueryName,
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

	testData := queryData{
		Language:            langDir,
		QueryPackName:       testPackFullName,
		QueryName:           opts.QueryName,
		QlLanguage:          langDir,
		QueryPackFullName:   testPackFullName,
		QueryPackDependency: packFullName,
		TestFilePrefix:      opts.QueryName,
	}
	testPackTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/qlpack-test.tmpl", langDir))
	if err != nil {
		return fmt.Errorf("load qlpack-test template: %w", err)
	}
	testPackPath := filepath.Join(opts.Base, langDir, packName, "test", "qlpack.yml")
	if err := tmpl.WriteFile(testPackTmpl, testPackPath, testData, opts.Overwrite); err != nil {
		return fmt.Errorf("write qlpack-test: %w", err)
	}

	slog.Info("Created test pack", "name", testPackFullName, "language", opts.Lang, "pack", opts.Pack)
	return nil
}
