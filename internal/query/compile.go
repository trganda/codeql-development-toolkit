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
	packpkg "github.com/trganda/codeql-development-toolkit/internal/pack"
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

// RunCompile compiles .ql files belonging to the selected packs using
// `codeql query compile`. When packs is empty, every pack listed under base is
// compiled; otherwise only packs whose full or unique short name matches an
// entry in packs are compiled.
func RunCompile(base string, packs []string, threads int) error {
	codeqlBin, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return err
	}

	cli := codeql.NewCLI(codeqlBin)
	allPacks, err := packpkg.ListPacks(cli, base)
	if err != nil {
		return fmt.Errorf("list packs: %w", err)
	}
	if len(allPacks) == 0 {
		return fmt.Errorf("no CodeQL packs found under %s", base)
	}

	selected, err := selectPacksForCompile(allPacks, packs)
	if err != nil {
		return err
	}

	var files []string
	for _, p := range selected {
		f, err := findQueryFiles(p.Dir())
		if err != nil {
			return fmt.Errorf("search query files in %s: %w", p.Dir(), err)
		}
		files = append(files, f...)
	}
	if len(files) == 0 {
		return fmt.Errorf("no .ql files found in selected packs")
	}

	maxWorkers := threads
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}

	slog.Info("Compiling query files", "count", len(files), "packs", len(selected), "workers", maxWorkers)
	for _, f := range files {
		slog.Debug("Scheduled for compile", "file", f)
	}

	var (
		mu   sync.Mutex
		errs []error
	)

	g := new(errgroup.Group)
	g.SetLimit(maxWorkers)

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

// selectPacksForCompile resolves the user-supplied pack name filter against
// the listed packs. An empty filter returns all packs unchanged. Names match
// by full name first, then by unique short name (segment after "/").
func selectPacksForCompile(allPacks []*packpkg.Pack, filter []string) ([]*packpkg.Pack, error) {
	if len(filter) == 0 {
		return allPacks, nil
	}

	selected := make([]*packpkg.Pack, 0, len(filter))
	for _, name := range filter {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		var (
			full         *packpkg.Pack
			shortMatches []*packpkg.Pack
		)
		for _, p := range allPacks {
			packName := p.Config.FullName()
			if packName == name {
				full = p
				break
			}
			if packpkg.GetPackName(packName) == name {
				shortMatches = append(shortMatches, p)
			}
		}
		if full != nil {
			selected = append(selected, full)
			continue
		}
		if len(shortMatches) == 1 {
			selected = append(selected, shortMatches[0])
			continue
		}
		if len(shortMatches) > 1 {
			var names []string
			for _, p := range shortMatches {
				names = append(names, p.Config.FullName())
			}
			return nil, fmt.Errorf("pack %q matches multiple packs; use full name from qlt pack list: %s",
				name, strings.Join(names, ", "))
		}
		return nil, fmt.Errorf("no pack matched %q under base (run qlt pack list)", name)
	}
	return selected, nil
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
