package pack

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/executil"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// packLsOutput mirrors the JSON structure of `codeql pack ls --format json`.
type packLsOutput struct {
	Packs map[string]struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"packs"`
}

// Entry pairs parsed pack metadata with the qlpack.yml directory path.
type Entry struct {
	Dir     string
	Name    string
	Version string
}

// FindQlpacks runs `codeql pack ls --format json <dir>` and returns all entries,
// optionally filtered by language directory and pack name segment.
func FindQlpacks(base, lang, pack string) ([]Entry, error) {
	searchDir := base
	if lang != "" {
		searchDir = filepath.Join(base, language.ToDirectory(lang))
	}

	codeql, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return nil, err
	}
	runner := executil.NewRunner(codeql)

	res, err := runner.Run("pack", "ls", "--format", "json", searchDir)
	if err != nil {
		if res != nil && len(res.Stderr) > 0 {
			slog.Debug("codeql pack ls stderr", "output", res.StderrString())
		}
		return nil, fmt.Errorf("run codeql pack ls: %w", err)
	}
	if len(res.Stderr) > 0 {
		slog.Debug("codeql pack ls stderr", "output", res.StderrString())
	}

	var output packLsOutput
	if err := json.Unmarshal(res.Stdout, &output); err != nil {
		return nil, fmt.Errorf("parse codeql pack ls output: %w", err)
	}

	var entries []Entry
	for qlpackFile, meta := range output.Packs {
		if meta.Name == "" {
			continue
		}
		// Pack filter: match against the segment after the last '/'.
		if pack != "" {
			segments := strings.SplitN(meta.Name, "/", 2)
			packSegment := segments[len(segments)-1]
			if !strings.EqualFold(packSegment, pack) {
				continue
			}
		}
		entries = append(entries, Entry{
			Dir:     filepath.Dir(qlpackFile),
			Name:    meta.Name,
			Version: meta.Version,
		})
	}
	return entries, nil
}
