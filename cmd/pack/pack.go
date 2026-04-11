package pack

import (
	"log/slog"

	"github.com/spf13/cobra"
)

// NewCommand returns the `pack` cobra command.
func NewCommand(base, automationType *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack",
		Short: "CodeQL pack management commands",
	}
	cmd.AddCommand(newListCmd(base))
	cmd.AddCommand(newRunCmd(base))
	return cmd
}

func newRunCmd(base *string) *cobra.Command {
	run := &cobra.Command{
		Use:   "run",
		Short: "Run pack commands",
	}
	run.AddCommand(newHelloCmd(base))
	return run
}

func newHelloCmd(base *string) *cobra.Command {
	return &cobra.Command{
		Use:   "hello",
		Short: "Pack hello command (placeholder)",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing pack run hello command", "base", *base)
			slog.Info("Pack command — coming soon")
			return nil
		},
	}
}
