package pack

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/config"
)

func newSetCmd(base string) *cobra.Command {
	var (
		bundle           bool
		publish          bool
		referencesBundle bool
	)

	cmd := &cobra.Command{
		Use:   "set <pack-name>",
		Short: "Set configuration for a CodeQL pack in qlt.conf.json",
		Long: `Set or update configuration fields for a CodeQL pack entry.

If the pack does not yet exist in qlt.conf.json, it will be created.
Only the flags you provide will be changed; other fields are left as-is.

Examples:
  qlt pack set scope/my-pack --bundle
  qlt pack set scope/my-pack --bundle=false --publish
  qlt pack set scope/my-pack --references-bundle`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			slog.Debug("Executing pack set command", "base", base, "pack", name)

			cfg := config.MustLoadFromFile(base)

			entry, idx := findPack(cfg, name)
			if entry == nil {
				return fmt.Errorf("pack %q not found in qlt.conf.json; add it first via 'qlt query generate new-query'", name)
			}

			if cmd.Flags().Changed("bundle") {
				entry.Bundle = bundle
			}
			if cmd.Flags().Changed("publish") {
				entry.Publish = publish
			}
			if cmd.Flags().Changed("references-bundle") {
				entry.ReferencesBundle = referencesBundle
			}

			cfg.CodeQLPackConfiguration[idx] = *entry

			if err := cfg.SaveToFile(base); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Printf("Updated pack %q in qlt.conf.json\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVar(&bundle, "bundle", false, "Include this pack in custom bundle creation")
	cmd.Flags().BoolVar(&publish, "publish", false, "Include this pack in publish phase")
	cmd.Flags().BoolVar(&referencesBundle, "references-bundle", false, "This pack references a custom bundle")

	return cmd
}

func findPack(cfg *config.QLTConfig, name string) (*config.CodeQLPackConfiguration, int) {
	for i := range cfg.CodeQLPackConfiguration {
		if cfg.CodeQLPackConfiguration[i].Name == name {
			return &cfg.CodeQLPackConfiguration[i], i
		}
	}
	return nil, -1
}
