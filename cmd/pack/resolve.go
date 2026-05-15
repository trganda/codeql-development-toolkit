package pack

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/pack"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

func newResolveCmd(base *string) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Sync qlt.conf.json with the packs discovered under <base>",
		Long: `Scan <base> for CodeQL packs, exclude test packs, add any newly
discovered packs to qlt.conf.json, and remove entries that no longer
correspond to a discovered pack.`,
		Run: func(cmd *cobra.Command, args []string) {
			slog.Debug("Executing pack resolve command", "base", *base)
			if err := runPackResolve(*base); err != nil {
				slog.Error("Pack resolve failed", "base", *base, "err", err)
				os.Exit(1)
			}
		},
	}

	return cmd
}

func runPackResolve(base string) error {
	codeqlBin, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return fmt.Errorf("resolve CodeQL binary: %w", err)
	}
	slog.Debug("Resolved CodeQL binary", "path", codeqlBin)

	packs, err := pack.ListPacks(codeql.NewCLI(codeqlBin), base)
	if err != nil {
		return fmt.Errorf("list packs: %w", err)
	}

	cfg := config.MustLoadFromFile(base)

	discovered := make(map[string]struct{}, len(packs))
	added := 0
	for _, p := range packs {
		if p.IsTestPack() {
			slog.Debug("Skipping test pack", "name", p.Config.FullName())
			continue
		}
		name := p.Config.FullName()
		discovered[name] = struct{}{}
		if packConfigExists(cfg, name) {
			slog.Debug("Pack already registered", "name", name)
			continue
		}
		cfg.UpsertPackConfig(name, false)
		fmt.Printf("Added %s\n", name)
		added++
	}

	removed := 0
	for _, name := range packConfigNames(cfg) {
		if _, ok := discovered[name]; ok {
			continue
		}
		if cfg.RemovePackConfig(name) {
			fmt.Printf("Removed %s\n", name)
			removed++
		}
	}

	if added == 0 && removed == 0 {
		fmt.Println("qlt.conf.json already matches discovered packs.")
		return nil
	}

	if err := cfg.SaveToFile(base); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Printf("Synced qlt.conf.json: %d added, %d removed.\n", added, removed)
	return nil
}

func packConfigExists(cfg *config.QLTConfig, name string) bool {
	for _, p := range cfg.CodeQLPackConfiguration {
		if p.Name == name {
			return true
		}
	}
	return false
}

// packConfigNames returns a snapshot of the configured pack names; the snapshot
// lets the caller mutate the underlying slice (via RemovePackConfig) while iterating.
func packConfigNames(cfg *config.QLTConfig) []string {
	names := make([]string, 0, len(cfg.CodeQLPackConfiguration))
	for _, p := range cfg.CodeQLPackConfiguration {
		names = append(names, p.Name)
	}
	return names
}
