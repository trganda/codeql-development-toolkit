package bundle

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/archive"
	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/utils"
)

type CustomBundle struct {
	opts            *CreateOptions
	tmpCodeQLCLI    *codeql.CLI
	tmpBundleDir    string
	tmpQlPacksDir   string
	tmpDir          string
	commonCachesDir string
}

func NewCustomBundle(opts *CreateOptions) (*CustomBundle, error) {
	tmpDir, err := os.MkdirTemp("", "qlt-bundle-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() {
		slog.Debug("Removing temp dir", "path", tmpDir)
		os.RemoveAll(tmpDir)
	}()

	commonCaches := filepath.Join(tmpDir, "common-caches")
	// Create for common cache
	os.Mkdir(commonCaches, 0755)

	bundleDir := filepath.Join(tmpDir, "codeql")
	codeqlBin := filepath.Join(bundleDir, "codeql")
	if runtime.GOOS == "windows" {
		codeqlBin = filepath.Join(bundleDir, "codeql.exe")
	}

	return &CustomBundle{
		opts:            opts,
		tmpDir:          tmpDir,
		tmpBundleDir:    bundleDir,
		tmpQlPacksDir:   filepath.Join(bundleDir, "qlpacks"),
		tmpCodeQLCLI:    codeql.NewCLI(codeqlBin),
		commonCachesDir: commonCaches,
	}, nil
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
func (ctx *CustomBundle) Create() error {
	slog.Info("Extracting base bundle", "archive", ctx.opts.BundlePath)
	if err := archive.ExtractZip(ctx.opts.BundlePath, ctx.tmpDir); err != nil {
		return fmt.Errorf("extracting bundle: %w", err)
	}

	slog.Info("Listing workspace packs", "dir", ctx.opts.WorkspaceDir)

	if len(ctx.opts.Packs) == 0 {
		return fmt.Errorf("no pack found in workspace")
	}

	for _, p := range ctx.opts.Packs {
		if p.Config.Scope() == "" {
			return fmt.Errorf("pack %q has no scope; all bundled packs must be scoped", p.Config.FullName())
		}
	}

	// Clear exists qlpacks directory
	os.RemoveAll(ctx.tmpQlPacksDir)
	os.Mkdir(ctx.tmpQlPacksDir, 0755)

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

			filter := MakePlatformFilter(platform, languages)
			if err := archive.CreateTarGz(outFile, ctx.tmpBundleDir, "codeql", filter); err != nil {
				return fmt.Errorf("creating bundle archive for %s: %w", platform, err)
			}
		}
	}
	return nil
}

// MakePlatformFilter returns a filter function for CreateTarGz that excludes
// paths not belonging to the target platform.
func MakePlatformFilter(platform string, languages []string) func(string) bool {
	exclusions := platformExclusions(platform, languages)
	return func(relSlash string) bool {
		for _, excl := range exclusions {
			if relSlash == excl || strings.HasPrefix(relSlash, excl+"/") {
				return false
			}
		}
		return true
	}
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

// platformExclusions returns the set of slash-prefixed path prefixes that
// should be excluded for the given target platform. Each entry is relative to
// the bundle root (no leading slash).
//
// platform is one of "linux64", "osx64", "win64".
// languages is the set of language names returned by `codeql resolve languages`.
func platformExclusions(platform string, languages []string) []string {
	// Tools subdirectories per platform.
	linuxSubdirs := []string{"linux64", "linux"}
	osxSubdirs := []string{"osx64", "macos"}
	winSubdirs := []string{"win64", "windows"}

	var excludeSubdirs []string
	switch platform {
	case "linux64":
		excludeSubdirs = append(osxSubdirs, winSubdirs...)
	case "osx64":
		excludeSubdirs = append(linuxSubdirs, winSubdirs...)
	case "win64":
		excludeSubdirs = append(linuxSubdirs, osxSubdirs...)
	}

	// Base tools paths to filter.
	toolsPaths := []string{"tools"}
	for _, lang := range languages {
		toolsPaths = append(toolsPaths, lang+"/tools")
	}

	var exclusions []string
	for _, base := range toolsPaths {
		for _, sub := range excludeSubdirs {
			exclusions = append(exclusions, base+"/"+sub)
		}
	}

	// Per-platform binary exclusions.
	if platform != "win64" {
		exclusions = append(exclusions, "codeql.exe")
	}
	if platform == "win64" {
		exclusions = append(exclusions, "swift/qltest", "swift/resource-dir")
	}
	if platform == "linux64" {
		exclusions = append(exclusions, "swift/qltest/osx64", "swift/resource-dir/osx64")
	}
	if platform == "osx64" {
		exclusions = append(exclusions, "swift/qltest/linux64", "swift/resource-dir/linux64")
	}

	return exclusions
}
