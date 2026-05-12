package pack

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	packsvc "github.com/trganda/codeql-development-toolkit/internal/pack"
)

// newRunCmd returns `pack run` — analyze a CodeQL database using all queries in a pack.
func newRunCmd(base *string) *cobra.Command {
	var (
		database string
		packName string
		format   string
		output   string
		threads  int
	)
	run := &cobra.Command{
		Use:   "run",
		Short: "Analyze a CodeQL database using a query pack",
		Long: `Run 'codeql database analyze' against <database>, using every query in the resolved pack directory.

The pack is resolved the same way as qlt pack list (codeql pack ls under --base): use the full pack name from that list,
or an unambiguous short name (segment after '/').

If --output is omitted, results are written to <base>/target/analyze/<pack>.<ext>.

The CodeQL binary is resolved the same way as other qlt commands (bundle, ~/.qlt/packages, PATH).`,
		Run: func(cmd *cobra.Command, args []string) {
			slog.Debug("Executing pack run",
				"database", database, "pack", packName,
				"format", format, "output", output, "threads", threads)
			if err := packsvc.RunAnalyze(*base, database, packName, format, output, threads); err != nil {
				slog.Error("Pack run failed", "pack", packName, "database", database, "err", err)
				os.Exit(1)
			}
		},
	}
	run.Flags().StringVar(&database, "database", "", "Path to the CodeQL database (required)")
	run.Flags().StringVar(&packName, "pack", "", "Pack name as shown by qlt pack list (full name or unique short name, required)")
	run.Flags().StringVar(&format, "format", "sarif-latest", "Output format: sarif-latest, csv, text, dot, bqrs")
	run.Flags().StringVar(&output, "output", "", "Output file path (default: <base>/target/analyze/<pack>.<ext>)")
	run.Flags().IntVar(&threads, "threads", 0, "Number of threads (0 = use all available cores)")
	run.MarkFlagRequired("database")
	run.MarkFlagRequired("pack")
	return run
}
