package query

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	"github.com/trganda/codeql-development-toolkit/internal/pack"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
	tmpl "github.com/trganda/codeql-development-toolkit/internal/template"
)

type GenerateQueryOptions struct {
	Base      string
	Name      string
	Language  string
	QueryKind string
	PackName  string
	Overwrite bool
}

// queryData holds template variables for query scaffolding.
type queryData struct {
	Language         string
	QueryPackName    string
	QueryName        string
	Description      string
	QlLanguageImport string
	QlLanguage       string
	QueryKind        string
}

func GenerateNewQuery(opts *GenerateQueryOptions) error {
	langDir := language.ToDirectory(opts.Language)
	langImport := language.ToImport(opts.Language)

	codeqlBin, err := paths.ResolveCodeQLBinary(opts.Base)
	if err != nil {
		return fmt.Errorf("resolve CodeQL binary: %w", err)
	}

	targetDir := filepath.Join(opts.Base)
	existing, err := pack.ListPacks(codeql.NewCLI(codeqlBin), targetDir)
	if err != nil {
		return fmt.Errorf("check existing packs: %w", err)
	}

	exists := false
	for _, p := range existing {
		if p.Config.FullName() == opts.PackName {
			exists = true
			break
		}
	}
	if !exists {
		return fmt.Errorf("pack %q not exists at %s; use pack generate to crate it first", opts.PackName, targetDir)
	}

	// Query file goes in <base>/<langDir>/<pack>/src/<queryName>/<queryName>.ql
	queryDir := filepath.Join(opts.Base, langDir, pack.GetPackName(opts.PackName), "src", opts.Name)
	if err := os.MkdirAll(queryDir, 0755); err != nil {
		return fmt.Errorf("create query directory: %w", err)
	}

	// Determine template name based on kind.
	tmplName := "new-query"
	if strings.ToLower(opts.QueryKind) == "path-problem" {
		tmplName = "new-dataflow-query"
	}

	slog.Debug("Creating new query", "language", opts.Language, "dir", langDir, "pack", opts.PackName, "kind", opts.QueryKind)

	data := queryData{
		Language:         langDir,
		QueryName:        opts.Name,
		QueryPackName:    pack.GetPackName(opts.PackName),
		Description:      "Replace this text with a description of your query.",
		QlLanguageImport: langImport,
		QlLanguage:       langDir,
		QueryKind:        opts.QueryKind,
	}

	// Write query file.
	queryTmpl, err := tmpl.Get(fmt.Sprintf("query/%s/%s.tmpl", langDir, tmplName))
	if err != nil {
		return fmt.Errorf("load query template: %w", err)
	}
	queryFilePath := filepath.Join(queryDir, opts.Name+".ql")
	if err := tmpl.WriteFile(queryTmpl, queryFilePath, data, opts.Overwrite); err != nil {
		return fmt.Errorf("write query file: %w", err)
	}

	return nil
}
