package query

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/executil"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// RunQuery resolves the query file and runs `codeql database analyze`.
func RunQuery(base, queryName, database, lang, pack, format, output, additionalPacks string, threads int) error {
	queryFile, err := resolveQueryFile(base, queryName, lang, pack)
	if err != nil {
		return err
	}
	slog.Debug("Using query file", "path", queryFile)

	if _, err := os.Stat(database); err != nil {
		return fmt.Errorf("database not found: %s", database)
	}

	if !isKnownLanguage(lang) {
		slog.Info("Unrecognised language, proceeding anyway", "language", lang)
	}

	codeql, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return err
	}
	slog.Debug("Using CodeQL binary", "path", codeql)

	if output == "" {
		ext := formatExtension(format)
		output = filepath.Join(filepath.Dir(queryFile), queryName+ext)
	}

	args := buildAnalyzeArgs(database, queryFile, format, output, additionalPacks, threads)
	slog.Debug("Running CodeQL", "cmd", codeql, "args", args)

	runner := executil.NewRunner(codeql)
	res, err := runner.Run(args...)
	if err != nil {
		if res != nil && len(res.Stdout) > 0 {
			slog.Debug("Command stdout result", "output", res.StdoutString())
		}
		return fmt.Errorf("run codeql: %w", err)
	}
	if len(res.Stdout) > 0 {
		slog.Debug("Command stdout result", "output", res.StdoutString())
	}
	if len(res.Stderr) > 0 {
		slog.Debug("Command stderr result", "output", res.StderrString())
	}

	slog.Info("Results written to ", "path", output)
	return nil
}

// resolveQueryFile finds the .ql file for queryName.
//
// Resolution order:
//  1. Filesystem search: walk up to 3 levels under <base>/<langDir>/[pack] looking
//     for <queryName>.ql. This covers queries created manually outside the generator.
func resolveQueryFile(base, queryName, lang, pack string) (string, error) {
	langDir := language.ToDirectory(lang)

	searchRoot := filepath.Join(base, langDir)
	if pack != "" {
		searchRoot = filepath.Join(searchRoot, pack)
	}
	found, err := findQueryFile(searchRoot, queryName+".ql", 3)
	if err != nil {
		return "", fmt.Errorf("query %q not found under %s: %w", queryName, searchRoot, err)
	}
	slog.Debug("Resolved query by filesystem search", "path", found)
	return found, nil
}

// findQueryFile recursively searches dir for a file named target up to maxDepth levels deep.
func findQueryFile(dir, target string, maxDepth int) (string, error) {
	var found string
	var search func(string, int)
	search = func(d string, depth int) {
		if found != "" {
			return
		}
		entries, err := os.ReadDir(d)
		if err != nil {
			return
		}
		for _, e := range entries {
			path := filepath.Join(d, e.Name())
			if !e.IsDir() && e.Name() == target {
				found = path
				return
			}
			if e.IsDir() && depth < maxDepth {
				search(path, depth+1)
			}
		}
	}
	search(dir, 0)
	if found == "" {
		return "", fmt.Errorf("not found")
	}
	return found, nil
}

func buildAnalyzeArgs(database, queryFile, format, output, additionalPacks string, threads int) []string {
	args := []string{
		"database", "analyze",
		"--format=" + format,
		"--output=" + output,
		fmt.Sprintf("--threads=%d", threads),
		"--rerun",
	}
	if additionalPacks != "" {
		args = append(args, "--additional-packs="+additionalPacks)
	}
	args = append(args, database, queryFile)
	return args
}

func formatExtension(format string) string {
	switch strings.ToLower(format) {
	case "sarif-latest", "sarifv2.1.0":
		return ".sarif"
	case "csv":
		return ".csv"
	case "dot":
		return ".dot"
	case "text":
		return ".txt"
	case "bqrs":
		return ".bqrs"
	default:
		return ".sarif"
	}
}

func isKnownLanguage(lang string) bool {
	known := map[string]bool{
		"c": true, "cpp": true, "csharp": true, "go": true,
		"java": true, "javascript": true, "python": true, "ruby": true,
		"swift": true, "kotlin": true,
	}
	return known[strings.ToLower(lang)]
}
