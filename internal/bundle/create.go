package bundle

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/trganda/codeql-development-toolkit/internal/archive"
	"github.com/trganda/codeql-development-toolkit/internal/executil"
)

// buildCtx holds shared state during pack processing.
type buildCtx struct {
	codeqlBin    string
	bundleDir    string
	qlpacksDir   string
	tmpDir       string
	commonCaches string
	workspaceDir string
}

// Create builds a custom CodeQL bundle by extending the base bundle with
// the configured workspace packs. The flow is:
//
//  1. Extract the base bundle into a temp directory.
//  2. Clear the bundle's qlpacks/ directory (a clean slate; stdlib deps are
//     restored in step 6).
//  3. For each workspace pack, copy it under <tmp>/temp and run
//     `codeql pack install --common-caches=<tmp>/common-caches` then
//     `codeql pack create --output=<qlpacksDir> --common-caches=<tmp>/common-caches`.
//  4. Copy <tmp>/common-caches/packages/* into <qlpacksDir> so the bundle
//     contains every resolved dependency.
//  5. Repack the modified bundle, either as a single archive or one per
//     requested platform.
//
// Only QueryPack workspace packs are processed in full. Customization and
// Library packs are skipped with a warning (future work).
func Create(opts CreateOptions) error {
	if err := opts.Validate(); err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "qlt-bundle-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() {
		slog.Debug("Removing temp dir", "path", tmpDir)
		os.RemoveAll(tmpDir)
	}()

	slog.Info("Extracting base bundle", "archive", opts.BundlePath)
	if err := archive.ExtractTarGz(opts.BundlePath, tmpDir); err != nil {
		return fmt.Errorf("extracting bundle: %w", err)
	}
	bundleDir := filepath.Join(tmpDir, "codeql")

	codeqlBin := filepath.Join(bundleDir, "codeql")
	if runtime.GOOS == "windows" {
		codeqlBin = filepath.Join(bundleDir, "codeql.exe")
	}
	if _, err := os.Stat(codeqlBin); err != nil {
		return fmt.Errorf("codeql binary not found in bundle at %s: %w", codeqlBin, err)
	}
	slog.Debug("Using bundle codeql binary", "path", codeqlBin)

	languages, err := resolveLanguages(codeqlBin)
	if err != nil {
		return fmt.Errorf("resolving languages: %w", err)
	}
	slog.Debug("Bundle languages", "languages", languages)

	slog.Info("Listing workspace packs", "dir", opts.WorkspaceDir)
	allWorkspacePacks, err := ListPacks(codeqlBin, opts.WorkspaceDir)
	if err != nil {
		return fmt.Errorf("listing workspace packs: %w", err)
	}
	selected, err := selectPacks(allWorkspacePacks, opts.Packs)
	if err != nil {
		return err
	}
	for _, p := range selected {
		if p.Config.Scope() == "" {
			return fmt.Errorf("pack %q has no scope; all bundled packs must be scoped", p.Config.Name)
		}
	}

	qlpacksDir := filepath.Join(bundleDir, "qlpacks")
	slog.Debug("Clearing qlpacks directory", "path", qlpacksDir)
	if err := os.RemoveAll(qlpacksDir); err != nil {
		return fmt.Errorf("clearing qlpacks dir: %w", err)
	}
	if err := os.MkdirAll(qlpacksDir, 0755); err != nil {
		return fmt.Errorf("creating qlpacks dir: %w", err)
	}

	ctx := &buildCtx{
		codeqlBin:    codeqlBin,
		bundleDir:    bundleDir,
		qlpacksDir:   qlpacksDir,
		tmpDir:       tmpDir,
		commonCaches: filepath.Join(tmpDir, "common-caches"),
		workspaceDir: opts.WorkspaceDir,
	}

	slog.Info("Processing packs", "count", len(selected))
	for _, p := range selected {
		if err := dispatchPack(ctx, p); err != nil {
			return fmt.Errorf("processing %s: %w", p.Config.Name, err)
		}
	}

	// Copy dependencies resolved by `pack install` into the bundle's qlpacks/.
	depsDir := filepath.Join(ctx.commonCaches, "packages")
	if _, err := os.Stat(depsDir); err == nil {
		slog.Info("Copying resolved dependencies into bundle", "from", depsDir, "to", qlpacksDir)
		if err := copyDir(depsDir, qlpacksDir); err != nil {
			return fmt.Errorf("copying resolved dependencies: %w", err)
		}
	} else {
		slog.Debug("No common-caches packages directory found; skipping dependency copy", "path", depsDir)
	}

	if len(opts.Platforms) == 0 {
		outputPath := opts.OutputPath
		if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
			outputPath = filepath.Join(outputPath, "codeql-bundle.tar.gz")
		}
		slog.Info("Creating platform-agnostic bundle", "output", outputPath)
		if err := CreateTarGz(outputPath, bundleDir, nil); err != nil {
			return fmt.Errorf("creating bundle archive: %w", err)
		}
	} else {
		if err := os.MkdirAll(opts.OutputPath, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
		for _, platform := range opts.Platforms {
			outFile := filepath.Join(opts.OutputPath, fmt.Sprintf("codeql-bundle-%s.tar.gz", platform))
			slog.Info("Creating platform-specific bundle", "platform", platform, "output", outFile)
			filter := MakePlatformFilter(platform, languages)
			if err := CreateTarGz(outFile, bundleDir, filter); err != nil {
				return fmt.Errorf("creating bundle archive for %s: %w", platform, err)
			}
		}
	}

	slog.Info("Custom bundle creation complete")
	return nil
}

// dispatchPack routes a pack to its per-kind processor.
func dispatchPack(ctx *buildCtx, p *Pack) error {
	switch p.Kind {
	case QueryPack:
		return processQueryPack(ctx, p)
	case CustomizationPack:
		return processCustomizationPack(ctx, p)
	case LibraryPack:
		return processLibraryPack(ctx, p)
	default:
		return fmt.Errorf("pack %s: unknown kind %v", p.Config.Name, p.Kind)
	}
}

// selectPacks filters workspacePacks to only the named packs. Returns an error
// if any requested pack is not found in the workspace.
func selectPacks(workspacePacks []*Pack, names []string) ([]*Pack, error) {
	byName := make(map[string]*Pack, len(workspacePacks))
	for _, p := range workspacePacks {
		byName[p.Config.Name] = p
	}
	var selected []*Pack
	for _, name := range names {
		p, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("pack %q not found in workspace", name)
		}
		selected = append(selected, p)
	}
	return selected, nil
}

// resolveLanguages runs `codeql resolve languages --format=json` and returns
// the language names (used for platform-specific archive filtering).
func resolveLanguages(codeqlBin string) ([]string, error) {
	res, err := executil.NewRunner(codeqlBin).Run("resolve", "languages", "--format=json")
	if err != nil {
		return nil, err
	}
	var langs map[string]any
	if err := json.Unmarshal(res.Stdout, &langs); err != nil {
		return nil, fmt.Errorf("parsing languages output: %w", err)
	}
	result := make([]string, 0, len(langs))
	for k := range langs {
		result = append(result, k)
	}
	return result, nil
}
