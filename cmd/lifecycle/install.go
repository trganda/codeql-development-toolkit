package lifecycle

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/query"
)

func newInstallCmd(base *string) *cobra.Command {
	var lang, packName string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install CodeQL pack dependencies",
		Long: `Install lifecycle phase: install dependencies for CodeQL packs.

Requires workspace initialization (run 'qlt lifecycle init' first).
Runs 'codeql pack install' for packs found under <base>, optionally filtered
by language and pack name.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing lifecycle install", "base", *base, "language", lang, "pack", packName)
			if err := checkWorkspace(*base); err != nil {
				return err
			}
			return runInstallStep(*base, lang, packName)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Filter by language (e.g. go, java)")
	cmd.Flags().StringVar(&packName, "pack", "", "Filter by pack name (exact match on the pack segment)")
	return cmd
}

// checkWorkspace returns an error if codeql-workspace.yml does not exist under base.
// All lifecycle phases except init call this as their first step.
func checkWorkspace(base string) error {
	if _, err := os.Stat(filepath.Join(base, "codeql-workspace.yml")); os.IsNotExist(err) {
		return fmt.Errorf("workspace not initialized — run 'qlt lifecycle init' first")
	}
	return nil
}

// runInstallStep warns if no qlpack.yml files exist, then runs pack install.
// Called by every lifecycle phase that chains through install.
func runInstallStep(base, lang, packName string) error {
	if !hasAnyQlpacks(base) {
		fmt.Println("No CodeQL packs found. Run 'qlt query generate new-query' to create your first query.")
	}
	return query.RunPackInstall(base, lang, packName)
}

// hasAnyQlpacks returns true if at least one qlpack.yml exists under base.
func hasAnyQlpacks(base string) bool {
	found := false
	_ = filepath.WalkDir(base, func(_ string, d fs.DirEntry, _ error) error {
		if !d.IsDir() && d.Name() == "qlpack.yml" {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
