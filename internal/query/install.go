package query

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	packpkg "github.com/trganda/codeql-development-toolkit/internal/pack"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// installPackEntry is one dependency entry in `codeql pack install --format=json` output.
type installPackEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Library bool   `json:"library"`
	Result  string `json:"result"` // "IGNORED" = already cached, "DOWNLOADED" = fetched
}

// installResult is the top-level `codeql pack install --format=json` output.
type installResult struct {
	Packs   []installPackEntry `json:"packs"`
	PackDir string             `json:"packDir"`
}

// resolveDepsResult is the `codeql pack resolve-dependencies --format=json` output:
// a flat map of pack name -> required version.
type resolveDepsResult map[string]string

// resolvePackItem is one entry inside the "found" map of `codeql resolve packs`.
type resolvePackItem struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

// resolvePackStep is one step in `codeql resolve packs --format=json` output.
// For "by-name-and-version" steps, Found is populated directly on the step.
type resolvePackStep struct {
	Type  string                                `json:"type"`
	Path  string                                `json:"path"`
	Found map[string]map[string]resolvePackItem `json:"found"`
}

// resolvePacksOutput is the top-level `codeql resolve packs --format=json` output.
type resolvePacksOutput struct {
	Steps []resolvePackStep `json:"steps"`
}

// RunPackInstall resolves qlpacks under the target path using `codeql pack ls`
// and runs `codeql pack install` for each pack whose dependencies are not fully
// cached. Deps are checked via `codeql pack resolve-dependencies` and
// `codeql resolve packs` before triggering an install.
func RunPackInstall(base string) error {
	codeqlBin, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return err
	}
	cli := codeql.NewCLI(codeqlBin)

	qlpacks, err := packpkg.ListPacks(cli, base)
	if err != nil {
		return err
	}
	if len(qlpacks) == 0 {
		return fmt.Errorf("No CodeQL packs found under %s. Run 'qlt query generate new-query' to create your first query.", base)
	}

	slog.Info("Found query packs under base", "base", base, "count", len(qlpacks))

	// Snapshot the local pack cache once for all packs.
	cached, err := resolvedPackCache(cli)
	if err != nil {
		// Non-fatal: fall back to always installing.
		slog.Debug("Could not resolve pack cache, skipping pre-check", "error", err)
		cached = nil
	}

	for _, p := range qlpacks {
		if cached != nil {
			allCached, err := allDepsCached(cli, p.Dir(), cached)
			if err != nil {
				slog.Debug("Could not resolve dependencies, proceeding with install", "qlpack", p.YmlPath, "error", err)
			} else if allCached {
				slog.Info("Skipping install, all deps cached", "qlpack", p.YmlPath)
				continue
			}
		}

		slog.Info("Installing pack dependencies", "qlpack", p.YmlPath)
		res, err := cli.PackInstall(p.YmlPath, "")
		if err != nil {
			if res != nil && len(res.Stdout) > 0 {
				slog.Debug("CodeQL pack install stdout", "qlpack", p.YmlPath, "output", res.StdoutString())
			}
			return fmt.Errorf("run codeql pack install for %s: %w", p.YmlPath, err)
		}
		if len(res.Stdout) > 0 {
			logInstallResult(p.YmlPath, res.Stdout)
		}
	}

	slog.Info("Installed dependencies for all query packs under target path", "targetPath", base, "count", len(qlpacks))
	return nil
}

// resolvedPackCache calls `codeql resolve packs` and returns a set of
// "name@version" strings for every pack found in the local cache
// ("by-name-and-version" steps).
func resolvedPackCache(cli *codeql.CLI) (map[string]struct{}, error) {
	res, err := cli.ResolvePacks()
	if err != nil {
		return nil, fmt.Errorf("resolve packs: %w", err)
	}

	var out resolvePacksOutput
	if err := json.Unmarshal(res.Stdout, &out); err != nil {
		return nil, fmt.Errorf("parse resolve packs output: %w", err)
	}

	cached := make(map[string]struct{})
	for _, step := range out.Steps {
		if step.Type != "by-name-and-version" {
			continue
		}
		for name, versions := range step.Found {
			for version := range versions {
				cached[name+"@"+version] = struct{}{}
			}
		}
	}
	return cached, nil
}

// allDepsCached returns true when every dependency required by the pack at dir
// is already present in the cached set.
func allDepsCached(cli *codeql.CLI, dir string, cached map[string]struct{}) (bool, error) {
	res, err := cli.PackResolveDependencies(dir)
	if err != nil {
		return false, fmt.Errorf("resolve dependencies for %s: %w", dir, err)
	}

	var deps resolveDepsResult
	if err := json.Unmarshal(res.Stdout, &deps); err != nil {
		return false, fmt.Errorf("parse resolve-dependencies output: %w", err)
	}

	for name, version := range deps {
		if _, ok := cached[name+"@"+version]; !ok {
			slog.Debug("Dep not cached", "dep", name, "version", version)
			return false, nil
		}
	}
	return true, nil
}

// logInstallResult parses `codeql pack install --format=json` output and logs
// one structured line per dependency.
func logInstallResult(qlpack string, data []byte) {
	var result installResult
	if err := json.Unmarshal(data, &result); err != nil {
		slog.Debug("Could not parse pack install output", "error", err)
		return
	}
	for _, entry := range result.Packs {
		slog.Info("Dep install result",
			"qlpack", qlpack,
			"dep", entry.Name,
			"version", entry.Version,
			"result", entry.Result,
		)
	}
}
