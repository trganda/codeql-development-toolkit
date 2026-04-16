package query

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

type compilePosition struct {
	FileName  string `json:"fileName"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"endLine"`
	EndColumn int    `json:"endColumn"`
}

type compileMessage struct {
	Severity string          `json:"severity"`
	Message  string          `json:"message"`
	Position compilePosition `json:"position"`
}

type compileResult struct {
	Query        string           `json:"query"`
	RelativeName string           `json:"relativeName"`
	QlHash       string           `json:"qlHash"`
	Success      bool             `json:"success"`
	Messages     []compileMessage `json:"messages"`
}

// RunCompile compiles all .ql files under the resolved search root using
// `codeql query compile`.
func RunCompile(base, lang, pack string, threads int) error {
	codeqlBin, err := paths.ResolveCodeQLBinary(base)
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

	maxWorkers := threads
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}

	slog.Info("Compiling query files", "count", len(files), "root", searchRoot, "workers", maxWorkers)
	for _, f := range files {
		slog.Debug("Scheduled for compile", "file", f)
	}

	var (
		mu   sync.Mutex
		errs []error
	)

	g := new(errgroup.Group)
	g.SetLimit(maxWorkers)

	cli := codeql.NewCLI(codeqlBin)
	for _, f := range files {
		f := f
		g.Go(func() error {
			res, err := cli.QueryCompile(1, f)
			if err != nil {
				if res != nil && len(res.Stdout) > 0 {
					slog.Debug("CodeQL compile stdout", "output", res.StdoutString())
				}
				mu.Lock()
				errs = append(errs, fmt.Errorf("compile %s: %w", f, err))
				mu.Unlock()
				return nil // let other goroutines continue
			}
			if len(res.Stdout) > 0 {
				mu.Lock()
				logCompileResults(res.Stdout)
				mu.Unlock()
			}
			return nil
		})
	}

	g.Wait()
	return errors.Join(errs...)
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

// logCompileResults parses JSON output from `codeql query compile` and logs
// the result and any diagnostic messages for each query file.
func logCompileResults(data []byte) {
	var results []compileResult
	if err := json.Unmarshal(data, &results); err != nil {
		slog.Debug("Could not parse compile output", "error", err)
		return
	}

	for _, r := range results {
		if r.Success {
			slog.Info("Compiled", "query", r.RelativeName)
		} else {
			slog.Error("Compile failed", "query", r.RelativeName)
		}
		for _, msg := range r.Messages {
			args := []any{
				"query", r.RelativeName,
				"message", msg.Message,
				"file", msg.Position.FileName,
				"line", msg.Position.Line,
				"column", msg.Position.Column,
			}
			switch strings.ToUpper(msg.Severity) {
			case "ERROR":
				slog.Error("Compile message", args...)
			case "WARNING":
				slog.Warn("Compile message", args...)
			default:
				slog.Info("Compile message", args...)
			}
		}
	}
}
