package lifecycle

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/executil"
	"github.com/trganda/codeql-development-toolkit/internal/pack"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
	"github.com/trganda/codeql-development-toolkit/internal/query"
	qlttest "github.com/trganda/codeql-development-toolkit/internal/test"
	"github.com/trganda/codeql-development-toolkit/internal/utils"
)

func newPublishCmd(base *string) *cobra.Command {
	var lang, packName string
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish CodeQL packs to the GitHub Package Registry",
		Long: `Publish lifecycle phase: publish CodeQL packs to the GitHub Package Registry.

Runs the full chain: install → compile → test → verify → publish.
Requires workspace initialization (run 'qlt lifecycle init' first).

Scans for packs under <base> (optionally filtered by --language and --pack)
and publishes each using 'codeql pack publish'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing lifecycle publish", "base", *base, "language", lang, "pack", packName)
			if err := utils.CheckWorkspace(*base); err != nil {
				return err
			}
			if err := query.RunPackInstall(*base, lang, packName); err != nil {
				return err
			}
			if err := query.RunCompile(*base, lang, packName, 0); err != nil {
				return err
			}
			if err := qlttest.RunUnitTests(*base, lang, "", 4); err != nil {
				return err
			}
			fmt.Println("verify: not yet fully implemented.")
			fmt.Println("Run 'qlt validation run check-queries --language <lang>' for available checks.")
			return runLifecyclePublish(*base, lang, packName)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Filter by language (e.g. go, java)")
	cmd.Flags().StringVar(&packName, "pack", "", "Filter by pack name (exact match on the pack segment)")
	return cmd
}

func runLifecyclePublish(base, lang, packName string) error {

	codeql, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return err
	}

	targetDir := base
	if lang != "" {
		targetDir = filepath.Join(targetDir, lang)
	}

	packs, err := pack.ListPacks(codeql, targetDir)
	if err != nil {
		return fmt.Errorf("list packs: %w", err)
	}

	runner := executil.NewRunner(codeql)
	for _, p := range packs {
		slog.Info("Publishing pack", "name", p.Config.FullName(), "version", p.Config.Version, "dir", p.Dir())
		res, err := runner.Run("pack", "publish", p.Dir())
		if err != nil {
			if res != nil && len(res.Stderr) > 0 {
				slog.Debug("codeql pack publish stderr", "pack", p.Config.FullName(), "output", res.StderrString())
			}
			return fmt.Errorf("publish %s: %w", p.Config.FullName(), err)
		}
		if len(res.Stdout) > 0 {
			slog.Debug("codeql pack publish stdout", "pack", p.Config.FullName(), "output", res.StdoutString())
		}
		slog.Info("Published", "name", p.Config.FullName(), "version", p.Config.Version)
	}
	return nil
}
