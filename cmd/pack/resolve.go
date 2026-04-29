package pack

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	"github.com/trganda/codeql-development-toolkit/internal/pack"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

func newResolveCmd(base *string) *cobra.Command {
	var lang string
	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Auto-discover non-test packs and register them in qlt.conf.json",
		Long: `Scan <base> for CodeQL packs, exclude test packs, and add any
newly discovered packs to qlt.conf.json. Existing entries are left unchanged.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if lang != "" && !language.IsSupported(lang) {
				return fmt.Errorf("--language must be one of %v, got %q", language.SupportedLanguages, lang)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing pack resolve command", "base", *base, "language", lang)
			return runPackResolve(*base, lang)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Limit discovery to a language subdirectory (e.g. go, java)")
	return cmd
}

func runPackResolve(base, lang string) error {
	targetDir := base
	if lang != "" {
		targetDir = filepath.Join(targetDir, lang)
	}

	codeqlBin, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return fmt.Errorf("resolve CodeQL binary: %w", err)
	}
	slog.Debug("Resolved CodeQL binary", "path", codeqlBin)

	packs, err := pack.ListPacks(codeql.NewCLI(codeqlBin), targetDir)
	if err != nil {
		return fmt.Errorf("list packs: %w", err)
	}

	cfg := config.MustLoadFromFile(base)

	added := 0
	for _, p := range packs {
		if p.IsTestPack() {
			slog.Debug("Skipping test pack", "name", p.Config.FullName())
			continue
		}
		if packConfigExists(cfg, p.Config.FullName()) {
			slog.Debug("Pack already registered", "name", p.Config.FullName())
			continue
		}
		cfg.UpsertPackConfig(p.Config.FullName(), false)
		fmt.Printf("Added %s\n", p.Config.FullName())
		added++
	}

	if added == 0 {
		fmt.Println("No new packs found.")
		return nil
	}

	if err := cfg.SaveToFile(base); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Printf("Registered %d pack(s) in qlt.conf.json.\n", added)
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
