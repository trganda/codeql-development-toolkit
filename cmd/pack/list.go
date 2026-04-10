package pack

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

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

// qlpackEntry pairs parsed metadata with the qlpack.yml directory path.
type qlpackEntry struct {
	dir     string
	name    string
	version string
}

func newListCmd(base *string) *cobra.Command {
	var lang, pack string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List CodeQL packs under the base directory",
		Long: `List all CodeQL packs found under <base> using 'codeql pack ls'.

Use --language and --pack to narrow the search to a specific language directory
or pack name.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing pack list command", "base", *base, "language", lang, "pack", pack)
			return runPackList(*base, lang, pack)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Filter by language (e.g. go, java)")
	cmd.Flags().StringVar(&pack, "pack", "", "Filter by pack name (exact match on the pack segment)")
	return cmd
}

func runPackList(base, lang, pack string) error {
	entries, err := findQlpacks(base, lang, pack)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("No CodeQL packs found.")
		return nil
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return fmt.Errorf("resolve base path: %w", err)
	}
	for _, e := range entries {
		rel, err := filepath.Rel(absBase, e.dir)
		if err != nil {
			rel = e.dir
		}
		fmt.Printf("%-40s  %s  (%s)\n", e.name, e.version, rel)
	}
	return nil
}

// findQlpacks runs `codeql pack ls --format json <dir>` and returns all entries,
// optionally filtered by language directory and pack name segment.
func findQlpacks(base, lang, pack string) ([]qlpackEntry, error) {
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

	var entries []qlpackEntry
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
		entries = append(entries, qlpackEntry{
			dir:     filepath.Dir(qlpackFile),
			name:    meta.Name,
			version: meta.Version,
		})
	}
	return entries, nil
}
