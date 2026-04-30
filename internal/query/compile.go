package query

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/pack"
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

// RunCompile compiles qlpack belonging to the selected packs using
// `codeql query compile`. When packs is empty, every pack listed under base is
// compiled; otherwise only packs whose full or unique short name matches an
// entry in packs are compiled.
func RunCompile(base string, packs []string, threads int) error {
	codeqlBin, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return err
	}

	cli := codeql.NewCLI(codeqlBin)
	allPacks, err := pack.ListPacks(cli, base)
	if err != nil {
		return fmt.Errorf("list packs: %w", err)
	}
	if len(allPacks) == 0 {
		return fmt.Errorf("no CodeQL packs found under %s", base)
	}

	selected, err := pack.SelectPacks(allPacks, packs, false)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		return fmt.Errorf("no packs selected for compile")
	}

	maxWorkers := threads
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}

	slog.Info("Compiling query pack", "packs", len(selected), "workers", maxWorkers)

	var (
		mu   sync.Mutex
		errs []error
	)

	g := new(errgroup.Group)
	g.SetLimit(maxWorkers)

	for _, p := range selected {
		g.Go(func() error {
			res, err := cli.QueryCompile(1, p.Dir())
			if err != nil {
				if res != nil && len(res.Stdout) > 0 {
					slog.Debug("CodeQL compile stdout", "output", res.StdoutString())
				}
				mu.Lock()
				errs = append(errs, fmt.Errorf("compile %s: %w", p.Config.FullName(), err))
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
