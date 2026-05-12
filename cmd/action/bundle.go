package action

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	"github.com/trganda/codeql-development-toolkit/internal/pack"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
	tmpl "github.com/trganda/codeql-development-toolkit/internal/template"
)

func newInitBundleTestCmd(base *string) *cobra.Command {
	var (
		langs             []string
		branch            string
		overwriteExisting bool
	)
	cmd := &cobra.Command{
		Use:   "bundle-test",
		Short: "Initialize bundle support",
		PreRun: func(cmd *cobra.Command, args []string) {
			for _, l := range langs {
				if !language.IsSupported(l) {
					slog.Error("Invalid --language", "supported", language.SupportedLanguages, "got", l)
					os.Exit(1)
				}
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			slog.Debug("Executing bundle init command", "base", *base, "languages", langs, "branch", branch)
			if err := runBundleInit(*base, langs, branch, overwriteExisting); err != nil {
				slog.Error("Bundle init failed", "err", err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringArrayVar(&langs, "language", nil, "Language(s) for bundle (repeat for multiple, e.g. --language java --language cpp)")
	cmd.Flags().StringVar(&branch, "branch", "main", "Branch to trigger automation on")
	cmd.Flags().BoolVar(&overwriteExisting, "overwrite", false, "Overwrite existing files")
	cmd.MarkFlagRequired("language")

	return cmd
}

func runBundleInit(base string, langs []string, branch string, overwrite bool) error {
	slog.Debug("Running bundle init", "base", base, "langs", langs, "branch", branch)

	bundledPacks, err := loadBundledPackNames(base)
	if err != nil {
		return fmt.Errorf("load bundled packs from qlt.conf.json: %w", err)
	}

	data := tmpl.BundleInitOptions{
		Languages: langs,
		Branch:    branch,
		Packs:     bundledPacks,
	}

	// Write install-qlt action
	installTmpl, err := tmpl.Get("shared/actions/install-qlt.tmpl")
	if err != nil {
		return fmt.Errorf("load install-qlt template: %w", err)
	}
	installPath := filepath.Join(base, ".github", "actions", "install-qlt", "action.yml")
	if _, statErr := os.Stat(installPath); statErr == nil && !overwrite {
		slog.Info("Skipped file (already exists). Use --overwrite-existing to replace.", "path", installPath)
	}
	if err := tmpl.WriteFile(installTmpl, installPath, nil, overwrite); err != nil {
		return fmt.Errorf("write install-qlt: %w", err)
	}

	// Write bundle integration test workflow
	bundleTmpl, err := tmpl.Get("bundle/actions/run-bundle-integration-tests.tmpl")
	if err != nil {
		return fmt.Errorf("load bundle template: %w", err)
	}
	bundlePath := filepath.Join(base, ".github", "workflows", "run-bundle-integration-tests.yml")
	if _, statErr := os.Stat(bundlePath); statErr == nil && !overwrite {
		slog.Info("Skipped file (already exists). Use --overwrite-existing to replace.", "path", bundlePath)
	}
	if err := tmpl.WriteFile(bundleTmpl, bundlePath, data, overwrite); err != nil {
		return fmt.Errorf("write bundle workflow: %w", err)
	}

	slog.Info("Bundle integration test workflow initialized", "base", base)
	return nil
}

// loadBundledPackNames returns the names of pack entries in qlt.conf.json with
// Bundle: true, restricted to query packs. Library packs and test packs are
// filtered out, since github/codeql-action/init's `packs:` input runs queries
// (or applies data extensions) and library packs do neither.
func loadBundledPackNames(base string) ([]string, error) {
	cfg := config.MustLoadFromFile(base)

	var bundled []string
	for _, p := range cfg.CodeQLPackConfiguration {
		if p.Bundle {
			bundled = append(bundled, p.Name)
		}
	}
	if len(bundled) == 0 {
		return nil, nil
	}

	codeqlBin, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return nil, fmt.Errorf("resolve codeql binary: %w", err)
	}
	allPacks, err := pack.ListPacks(codeql.NewCLI(codeqlBin), base)
	if err != nil {
		return nil, fmt.Errorf("list packs under %s: %w", base, err)
	}
	byName := make(map[string]*pack.Pack, len(allPacks))
	for _, p := range allPacks {
		byName[p.Config.FullName()] = p
	}

	var names []string
	for _, name := range bundled {
		p, ok := byName[name]
		if !ok {
			slog.Warn("Bundled pack not found under base; skipping in workflow config", "pack", name)
			continue
		}
		if p.Config.Library {
			slog.Info("Skipping library pack in codeql-action packs config", "pack", name)
			continue
		}
		if p.IsTestPack() {
			slog.Info("Skipping test pack in codeql-action packs config", "pack", name)
			continue
		}
		if p.IsCustomizable() {
			slog.Info("Skipping customizable pack in codeql-action packs config", "pack", name)
			continue
		}
		names = append(names, name)
	}
	return names, nil
}
