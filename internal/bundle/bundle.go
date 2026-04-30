package bundle

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/trganda/codeql-development-toolkit/internal/archive"
	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/utils"
)

type CustomBundle struct {
	opts            *CreateOptions
	tmpDir          string
	tmpBundleDir    string
	tmpQlPacksDir   string
	tmpCodeQLCLI    *codeql.CLI
	commonCachesDir string
}

func NewCustomBundle(opts *CreateOptions, tmpDir string) *CustomBundle {
	tmpBundleDir := filepath.Join(tmpDir, "codeql")
	tmpQlPacksDir := filepath.Join(tmpBundleDir, "qlpacks")
	commonCachesDir := filepath.Join(tmpDir, "common-caches")
	codeqlBin := filepath.Join(tmpBundleDir, "codeql")
	if runtime.GOOS == "windows" {
		codeqlBin = filepath.Join(tmpBundleDir, "codeql.exe")
	}

	tmpCodeQLCLI := codeql.NewCLI(codeqlBin)

	return &CustomBundle{
		opts:            opts,
		tmpDir:          tmpDir,
		tmpBundleDir:    tmpBundleDir,
		tmpQlPacksDir:   tmpQlPacksDir,
		tmpCodeQLCLI:    tmpCodeQLCLI,
		commonCachesDir: commonCachesDir,
	}
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
// Customization packs are skipped with a warning (future work).
func (ctx *CustomBundle) Create() error {

	slog.Info("Extracting base bundle", "archive", ctx.opts.BundlePath)
	if err := archive.ExtractZip(ctx.opts.BundlePath, ctx.tmpDir); err != nil {
		return fmt.Errorf("extracting bundle: %w", err)
	}

	slog.Info("Listing workspace packs", "dir", ctx.opts.WorkspaceDir)

	if len(ctx.opts.Packs) == 0 {
		slog.Warn("No packs configured for bundling;")
		return nil
	}

	for _, p := range ctx.opts.Packs {
		if p.Config.Scope() == "" {
			return fmt.Errorf("pack %q has no scope; all bundled packs must be scoped", p.Config.FullName())
		}
	}

	if err := os.RemoveAll(ctx.tmpQlPacksDir); err != nil {
		return fmt.Errorf("clearing qlpacks dir: %w", err)
	}

	for _, p := range ctx.opts.Packs {
		processor := newPackProcessor(ctx, classify(p))
		if err := processor.Process(p); err != nil {
			return fmt.Errorf("processing pack %q: %w", p.Config.FullName(), err)
		}
	}

	// Copy dependencies resolved by `pack install` into the bundle's qlpacks/.
	depsDir := filepath.Join(ctx.commonCachesDir, "packages")
	if _, err := os.Stat(depsDir); err == nil {
		slog.Info("Copying resolved dependencies into bundle", "from", depsDir, "to", ctx.tmpQlPacksDir)
		if err := utils.CopyDir(depsDir, ctx.tmpQlPacksDir); err != nil {
			return fmt.Errorf("copying resolved dependencies: %w", err)
		}
	} else {
		slog.Debug("No common-caches packages directory found; skipping dependency copy", "path", depsDir)
	}

	if len(ctx.opts.Platforms) == 0 {
		outputPath := filepath.Join(ctx.opts.OutputPath, "codeql-bundle.tar.gz")
		slog.Info("Creating platform-agnostic bundle", "output", outputPath)

		if err := archive.CreateTarGz(outputPath, ctx.tmpBundleDir, "codeql", nil); err != nil {
			return fmt.Errorf("creating bundle archive: %w", err)
		}
	} else {
		languages, err := ctx.resolveLanguages()
		if err != nil {
			return fmt.Errorf("resolving languages: %w", err)
		}

		for _, platform := range ctx.opts.Platforms {
			outFile := filepath.Join(ctx.opts.OutputPath, fmt.Sprintf("codeql-bundle-%s.tar.gz", platform))
			slog.Info("Creating platform-specific bundle", "platform", platform, "output", outFile)

			filter := makePlatformFilter(platform, languages)
			if err := archive.CreateTarGz(outFile, ctx.tmpBundleDir, "codeql", filter); err != nil {
				return fmt.Errorf("creating bundle archive for %s: %w", platform, err)
			}
		}
	}
	return nil
}

func (ctx *CustomBundle) resolveLanguages() ([]string, error) {
	res, err := ctx.tmpCodeQLCLI.ResolveLanguages()
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
