package pack

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/pack"
)

func newListCmd(base *string) *cobra.Command {
	var lang, packName string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List CodeQL packs under the base directory",
		Long: `List all CodeQL packs found under <base> using 'codeql pack ls'.

Use --language and --pack to narrow the search to a specific language directory
or pack name.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing pack list command", "base", *base, "language", lang, "pack", packName)
			return runPackList(*base, lang, packName)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Filter by language (e.g. go, java)")
	cmd.Flags().StringVar(&packName, "pack", "", "Filter by pack name (exact match on the pack segment)")
	return cmd
}

func runPackList(base, lang, packName string) error {
	entries, err := pack.FindQlpacks(base, lang, packName)
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
		rel, err := filepath.Rel(absBase, e.Dir)
		if err != nil {
			rel = e.Dir
		}
		fmt.Printf("%-40s  %s  (%s)\n", e.Name, e.Version, rel)
	}
	return nil
}
