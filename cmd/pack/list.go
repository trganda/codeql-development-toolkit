package pack

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/pack"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

func newListCmd(base *string) *cobra.Command {

	var all bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List CodeQL packs under the base directory",
		Long:  `List all CodeQL packs found under <base> using 'codeql pack ls'. By default, test packs are excluded; use --all to include them.`,
		Run: func(cmd *cobra.Command, args []string) {
			slog.Debug("Executing pack list command", "base", *base, "all", all)
			if err := runPackList(*base, all); err != nil {
				slog.Error("Pack list failed", "err", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Include test packs in the listing")
	return cmd
}

func runPackList(base string, all bool) error {

	codeqlBin, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return fmt.Errorf("resolve CodeQL binary: %w", err)
	}
	slog.Debug("Resolved CodeQL binary", "path", codeqlBin)

	packs, err := pack.ListPacks(codeql.NewCLI(codeqlBin), base)
	if err != nil {
		return fmt.Errorf("list packs: %w", err)
	}

	absBase, err := filepath.Abs(base)
	if err != nil {
		return fmt.Errorf("resolve base path: %w", err)
	}
	for _, p := range packs {
		if !all && p.IsTestPack() {
			continue
		}
		rel, err := filepath.Rel(absBase, p.Dir())
		if err != nil {
			rel = p.Dir()
		}
		slog.Info("Found pack", "name", p.Config.FullName(), "version", p.Config.Version, "relativeDir", rel)
	}
	return nil
}
