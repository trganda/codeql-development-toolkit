package bundle

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/trganda/codeql-development-toolkit/internal/executil"
)

// processQueryPack runs the 6-step flow for a single query-kind pack:
//  1. Copy the workspace pack into <tmpDir>/temp/<scope>/<name>/<version>.
//  2. Run `codeql pack install --common-caches=<commonCaches>` on the copy.
//  3. Run `codeql pack create --output=<qlpacksDir> --common-caches=<commonCaches>`
//     on the copy.
//
// Dependencies pulled into <commonCaches>/packages are copied into <qlpacksDir>
// by the caller after all workspace packs have been processed.
func processQueryPack(ctx *buildCtx, p *Pack) error {
	slog.Info("Processing query pack", "pack", p.Config.Name)

	packCopy, err := copyPackTo(p, filepath.Join(ctx.tmpDir, "temp"))
	if err != nil {
		return err
	}

	runner := executil.NewRunner(ctx.codeqlBin)

	if _, err := runner.Run(
		"pack", "install",
		"--format=json",
		fmt.Sprintf("--common-caches=%s", ctx.commonCaches),
		packCopy.Dir(),
	); err != nil {
		return fmt.Errorf("codeql pack install %s: %w", p.Config.Name, err)
	}

	if _, err := runner.Run(
		"pack", "create",
		"--format=json",
		fmt.Sprintf("--output=%s", ctx.qlpacksDir),
		fmt.Sprintf("--common-caches=%s", ctx.commonCaches),
		packCopy.Dir(),
	); err != nil {
		return fmt.Errorf("codeql pack create %s: %w", p.Config.Name, err)
	}

	return nil
}
