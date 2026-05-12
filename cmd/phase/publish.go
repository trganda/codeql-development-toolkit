package phase

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/pack"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
	"github.com/trganda/codeql-development-toolkit/internal/utils"
)

func newPublishCmd(base *string, common *utils.CommonFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "publish",
		Short: "Publish CodeQL packs to the GitHub Package Registry",
		Long: `Publish phase: publish CodeQL packs to the GitHub Package Registry.

Runs the full chain: install → compile → test → verify → publish.
Requires workspace initialization (run 'qlt phase init' first).

Scans for packs under <base> and publishes each using 'codeql pack publish'.`,
		Run: func(cmd *cobra.Command, args []string) {
			slog.Debug("Executing phase publish", "base", *base)
			if err := runVerifyChain(*base, common); err != nil {
				slog.Error("Phase publish failed during verify chain", "err", err)
				os.Exit(1)
			}
			if err := runPublish(*base); err != nil {
				slog.Error("Phase publish failed", "err", err)
				os.Exit(1)
			}
		},
	}
}

func runPublish(base string) error {
	codeqlBin, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return err
	}
	cli := codeql.NewCLI(codeqlBin)

	packs, err := pack.ListPacks(cli, base)
	if err != nil {
		return fmt.Errorf("list packs: %w", err)
	}

	for _, p := range packs {
		slog.Info("Publishing pack", "name", p.Config.FullName(), "version", p.Config.Version, "dir", p.Dir())
		res, err := cli.PackPublish(p.Dir())
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
