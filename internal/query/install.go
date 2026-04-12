package query

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/executil"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// RunPackInstall resolves query packs under the target path using
// `codeql resolve packs` and runs `codeql pack install` for each resolved
// qlpack.yml.
func RunPackInstall(base, lang, pack string) error {
	codeql, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return err
	}
	runner := executil.NewRunner(codeql)

	targetPath := base
	if lang != "" {
		langDir := language.ToDirectory(lang)
		targetPath = filepath.Join(base, langDir)
	}
	if pack != "" {
		targetPath = filepath.Join(targetPath, pack)
	}

	if _, err := os.Stat(targetPath); err != nil {
		return fmt.Errorf("target path not found: %s", targetPath)
	}

	qlpacks, err := resolveQueryQlpackFiles(runner, targetPath)
	if err != nil {
		return err
	}
	if len(qlpacks) == 0 {
		return fmt.Errorf("No CodeQL packs found under %s. Run 'qlt query generate new-query' to create your first query.", targetPath)
	}

	for _, qlpack := range qlpacks {
		slog.Info("Installing pack dependencies", "qlpack", qlpack)
		res, err := runner.Run("pack", "install", qlpack)
		if err != nil {
			if res != nil && len(res.Stdout) > 0 {
				slog.Debug("CodeQL pack install stdout", "qlpack", qlpack, "output", res.StdoutString())
			}
			return fmt.Errorf("run codeql pack install for %s: %w", qlpack, err)
		}
		if len(res.Stdout) > 0 {
			slog.Debug("CodeQL pack install stdout", "qlpack", qlpack, "output", res.StdoutString())
		}
		if len(res.Stderr) > 0 {
			slog.Debug("CodeQL pack install stderr", "qlpack", qlpack, "output", res.StderrString())
		}
	}

	slog.Info("Installed dependencies for all query packs under target path", "targetPath", targetPath, "count", len(qlpacks))
	return nil
}

type resolvedPacksOutput struct {
	Steps []struct {
		Scans []struct {
			Paths []string `json:"paths"`
			Found map[string]struct {
				Path string `json:"path"`
			} `json:"found"`
		} `json:"scans"`
	} `json:"steps"`
}

func resolveQueryQlpackFiles(runner *executil.Runner, targetPath string) ([]string, error) {
	res, err := runner.Run(
		"resolve", "packs",
		"--additional-packs", targetPath,
		"--format", "json",
	)
	if err != nil {
		if res != nil && len(res.Stdout) > 0 {
			slog.Debug("CodeQL resolve packs stdout", "output", res.StdoutString())
		}
		return nil, fmt.Errorf("run codeql resolve packs: %w", err)
	}

	if len(res.Stderr) > 0 {
		slog.Debug("CodeQL resolve packs stderr", "output", res.StderrString())
	}

	var output resolvedPacksOutput
	if err := json.Unmarshal(res.Stdout, &output); err != nil {
		return nil, fmt.Errorf("parse codeql resolve packs output: %w", err)
	}

	targetAbs, err := filepath.Abs(targetPath)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path for target %s: %w", targetPath, err)
	}
	targetAbs = filepath.Clean(targetAbs)

	foundSet := make(map[string]struct{})
	for _, step := range output.Steps {
		for _, scan := range step.Scans {
			includeScan := false
			for _, scanPath := range scan.Paths {
				scanAbs, err := filepath.Abs(scanPath)
				if err != nil {
					continue
				}
				scanAbs = filepath.Clean(scanAbs)
				if isSubpath(targetAbs, scanAbs) {
					includeScan = true
					break
				}
			}
			if !includeScan {
				continue
			}

			for _, entry := range scan.Found {
				if entry.Path == "" {
					continue
				}
				packFileAbs, err := filepath.Abs(entry.Path)
				if err != nil {
					continue
				}
				packFileAbs = filepath.Clean(packFileAbs)
				if strings.EqualFold(filepath.Base(packFileAbs), "qlpack.yml") && isSubpath(targetAbs, filepath.Dir(packFileAbs)) {
					foundSet[packFileAbs] = struct{}{}
				}
			}
		}
	}

	qlpacks := make([]string, 0, len(foundSet))
	for packFile := range foundSet {
		qlpacks = append(qlpacks, packFile)
	}
	sort.Strings(qlpacks)
	return qlpacks, nil
}

func isSubpath(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if parent == child {
		return true
	}
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
