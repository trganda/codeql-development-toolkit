package pack

import (
	"fmt"
	"log/slog"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing pack list command", "base", *base, "all", all)
			return runPackList(*base, all)
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
		fmt.Printf("%-40s  %s  (%s)\n", p.Config.FullName(), p.Config.Version, rel)
	}
	return nil
}
