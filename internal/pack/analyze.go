package pack

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// RunAnalyze runs `codeql database analyze` using the resolved query pack directory
// as the analysis target (all queries in that pack).
func RunAnalyze(base, database, packRef string, format, output string, threads int) error {
	if _, err := os.Stat(database); err != nil {
		return fmt.Errorf("database not found: %s", database)
	}

	codeqlBin, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return err
	}
	slog.Debug("Using CodeQL binary", "path", codeqlBin)

	if strings.TrimSpace(packRef) == "" {
		return fmt.Errorf("pack name is required")
	}

	cli := codeql.NewCLI(codeqlBin)
	packs, err := ListPacks(cli, base)
	if err != nil {
		return fmt.Errorf("list packs: %w", err)
	}
	selected, err := SelectPacks(packs, []string{packRef}, true)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		return fmt.Errorf("no query pack matched %q under base (run qlt pack list)", packRef)
	}
	p := selected[0]
	packDir := p.Dir()
	slog.Debug("Resolved pack directory for analyze", "path", packDir, "pack", p.Config.FullName())

	if output == "" {
		output = defaultPackAnalyzeOutput(base, p.Config.FullName(), format)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	opts := codeql.DatabaseAnalyzeOptions{
		Database:  database,
		QueryFile: packDir,
		Format:    format,
		Output:    output,
		Threads:   threads,
	}
	slog.Debug("Running CodeQL database analyze on pack", "cmd", codeqlBin, "opts", opts)

	res, err := codeql.NewCLI(codeqlBin).DatabaseAnalyze(opts)
	if err != nil {
		if res != nil && len(res.Stdout) > 0 {
			slog.Debug("Command stdout result", "output", res.StdoutString())
		}
		return fmt.Errorf("run codeql: %w", err)
	}
	if len(res.Stdout) > 0 {
		slog.Debug("Command stdout result", "output", res.StdoutString())
	}
	if len(res.Stderr) > 0 {
		slog.Debug("Command stderr result", "output", res.StderrString())
	}

	slog.Info("Results written", "path", output)
	return nil
}

func defaultPackAnalyzeOutput(base, pack string, format string) string {
	safe := strings.ReplaceAll(strings.ReplaceAll(pack, "/", "_"), `\`, "_")
	ext := analyzeFormatExtension(format)
	return filepath.Join(base, "target", "analyze", safe+ext)
}

func analyzeFormatExtension(format string) string {
	switch strings.ToLower(format) {
	case "sarif-latest", "sarifv2.1.0":
		return ".sarif"
	case "csv":
		return ".csv"
	case "dot":
		return ".dot"
	case "text":
		return ".txt"
	case "bqrs":
		return ".bqrs"
	default:
		return ".sarif"
	}
}
