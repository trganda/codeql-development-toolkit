package query

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/trganda/codeql-development-toolkit/internal/executil"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// RunCompile compiles all .ql files under the resolved search root using
// `codeql query compile`.
func RunCompile(base, lang, pack string, threads int) error {
	codeql, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return err
	}

	var searchRoot string
	if lang != "" && pack != "" {
		langDir := language.ToDirectory(lang)
		searchRoot = filepath.Join(base, langDir, pack, "src")
		if _, err := os.Stat(searchRoot); err != nil {
			return fmt.Errorf("directory not found: %s", searchRoot)
		}
	} else {
		searchRoot = base
	}

	files, err := findQueryFiles(searchRoot)
	if err != nil {
		return fmt.Errorf("search for query files: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no .ql files found under %s", searchRoot)
	}

	slog.Info("Compiling query files", "count", len(files), "root", searchRoot)

	args := []string{"query", "compile"}
	if threads != 0 {
		args = append(args, fmt.Sprintf("--threads=%d", threads))
	}
	args = append(args, "--")
	args = append(args, files...)

	runner := executil.NewRunner(codeql)
	res, err := runner.Run(args...)
	if err != nil {
		if res != nil && len(res.Stdout) > 0 {
			slog.Debug("CodeQL compile stdout", "output", res.StdoutString())
		}
		return fmt.Errorf("run codeql query compile: %w", err)
	}
	if len(res.Stderr) > 0 {
		slog.Debug("CodeQL compile stderr", "output", res.StderrString())
	}
	return nil
}

// findQueryFiles walks dir recursively and collects all .ql file paths.
func findQueryFiles(dir string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(d.Name()) == ".ql" {
			found = append(found, path)
		}
		return nil
	})
	return found, err
}
