package query

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/trganda/codeql-development-toolkit/internal/executil"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	packpkg "github.com/trganda/codeql-development-toolkit/internal/pack"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// RunPackInstall resolves qlpacks under the target path using `codeql pack ls`
// and runs `codeql pack install` for each resolved qlpack.yml.
func RunPackInstall(base, lang string) error {
	codeql, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return err
	}
	runner := executil.NewRunner(codeql)

	targetPath := base
	if lang != "" {
		targetPath = filepath.Join(base, language.ToDirectory(lang))
	}

	if _, err := os.Stat(targetPath); err != nil {
		return fmt.Errorf("target path not found: %s", targetPath)
	}

	qlpacks, err := packpkg.ListPacks(codeql, targetPath)
	if err != nil {
		return err
	}
	if len(qlpacks) == 0 {
		return fmt.Errorf("No CodeQL packs found under %s. Run 'qlt query generate new-query' to create your first query.", targetPath)
	}

	for _, p := range qlpacks {
		slog.Info("Installing pack dependencies", "qlpack", p.YmlPath)
		res, err := runner.Run("pack", "install", p.YmlPath)
		if err != nil {
			if res != nil && len(res.Stdout) > 0 {
				slog.Debug("CodeQL pack install stdout", "qlpack", p.YmlPath, "output", res.StdoutString())
			}
			return fmt.Errorf("run codeql pack install for %s: %w", p.YmlPath, err)
		}
		if len(res.Stdout) > 0 {
			slog.Debug("CodeQL pack install stdout", "qlpack", p.YmlPath, "output", res.StdoutString())
		}
		if len(res.Stderr) > 0 {
			slog.Debug("CodeQL pack install stderr", "qlpack", p.YmlPath, "output", res.StderrString())
		}
	}

	slog.Info("Installed dependencies for all query packs under target path", "targetPath", targetPath, "count", len(qlpacks))
	return nil
}
