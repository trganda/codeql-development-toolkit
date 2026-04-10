package pack

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/executil"
	"github.com/trganda/codeql-development-toolkit/internal/pack"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

func newPublishCmd(base *string) *cobra.Command {
	var lang, packName string
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish CodeQL packs to the GitHub Package Registry",
		Long: `Publish CodeQL packs to the GitHub Package Registry using 'codeql pack publish'.

Scans for qlpack.yml files under <base> (optionally filtered by --language and
--pack) and publishes each matching pack.

Authentication is read from the GITHUB_TOKEN environment variable or the
standard CodeQL registry configuration (~/.codeql/config).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing pack publish command", "base", *base, "language", lang, "pack", packName)
			return runPackPublish(*base, lang, packName)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Filter by language (e.g. go, java)")
	cmd.Flags().StringVar(&packName, "pack", "", "Filter by pack name (exact match on the pack segment)")
	return cmd
}

func runPackPublish(base, lang, packName string) error {
	entries, err := pack.FindQlpacks(base, lang, packName)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("no CodeQL packs found to publish")
	}

	codeql, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return err
	}
	runner := executil.NewRunner(codeql)

	for _, e := range entries {
		slog.Info("Publishing pack", "name", e.Name, "version", e.Version, "dir", e.Dir)
		res, err := runner.Run("pack", "publish", e.Dir)
		if err != nil {
			if res != nil && len(res.Stderr) > 0 {
				slog.Debug("codeql pack publish stderr", "pack", e.Name, "output", res.StderrString())
			}
			return fmt.Errorf("publish %s: %w", e.Name, err)
		}
		if len(res.Stdout) > 0 {
			slog.Debug("codeql pack publish stdout", "pack", e.Name, "output", res.StdoutString())
		}
		fmt.Printf("Published %s@%s\n", e.Name, e.Version)
	}
	return nil
}
