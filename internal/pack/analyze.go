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

	cli := codeql.NewCLI(codeqlBin)
	p, err := FindPackForAnalyze(cli, base, packRef)
	if err != nil {
		return err
	}
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

// FindPackForAnalyze lists packs under base (same discovery as qlt pack list) and selects a
// non-test pack matching packRef by full name or unique short name.
func FindPackForAnalyze(cli *codeql.CLI, base, packRef string) (*Pack, error) {
	packs, err := ListPacks(cli, base)
	if err != nil {
		return nil, fmt.Errorf("list packs: %w", err)
	}
	return SelectPackForAnalyze(packs, packRef)
}

// SelectPackForAnalyze picks the query pack identified by packRef from an already-listed set.
func SelectPackForAnalyze(packs []*Pack, packRef string) (*Pack, error) {
	packRef = strings.TrimSpace(packRef)
	if packRef == "" {
		return nil, fmt.Errorf("pack name is required")
	}

	var shortMatches []*Pack
	for _, p := range packs {
		if p.IsTestPack() {
			continue
		}
		name := p.Config.FullName()
		if name == packRef {
			return p, nil
		}
		if GetPackName(name) == packRef {
			shortMatches = append(shortMatches, p)
		}
	}

	if len(shortMatches) == 1 {
		return shortMatches[0], nil
	}
	if len(shortMatches) > 1 {
		var names []string
		for _, p := range shortMatches {
			names = append(names, p.Config.FullName())
		}
		return nil, fmt.Errorf("pack %q matches multiple packs; use full name from qlt pack list: %s",
			packRef, strings.Join(names, ", "))
	}

	return nil, fmt.Errorf("no query pack matched %q under base (run qlt pack list)", packRef)
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
