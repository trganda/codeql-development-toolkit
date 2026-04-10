package pack

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/executil"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

func newPublishCmd(base *string) *cobra.Command {
	var lang, pack string
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish CodeQL packs to the GitHub Package Registry",
		Long: `Publish CodeQL packs to the GitHub Package Registry using 'codeql pack publish'.

Scans for qlpack.yml files under <base> (optionally filtered by --language and
--pack) and publishes each matching pack.

Authentication is read from the GITHUB_TOKEN environment variable or the
standard CodeQL registry configuration (~/.codeql/config).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing pack publish command", "base", *base, "language", lang, "pack", pack)
			return runPackPublish(*base, lang, pack)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Filter by language (e.g. go, java)")
	cmd.Flags().StringVar(&pack, "pack", "", "Filter by pack name (exact match on the pack segment)")
	return cmd
}

func runPackPublish(base, lang, pack string) error {
	entries, err := findQlpacks(base, lang, pack)
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
		slog.Info("Publishing pack", "name", e.name, "version", e.version, "dir", e.dir)
		res, err := runner.Run("pack", "publish", e.dir)
		if err != nil {
			if res != nil && len(res.Stderr) > 0 {
				slog.Debug("codeql pack publish stderr", "pack", e.name, "output", res.StderrString())
			}
			return fmt.Errorf("publish %s: %w", e.name, err)
		}
		if len(res.Stdout) > 0 {
			slog.Debug("codeql pack publish stdout", "pack", e.name, "output", res.StdoutString())
		}
		fmt.Printf("Published %s@%s\n", e.name, e.version)
	}
	return nil
}
