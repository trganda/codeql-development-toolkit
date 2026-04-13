package bundle

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/archive"
	"github.com/trganda/codeql-development-toolkit/internal/executil"
)

// CreateOptions controls how the custom bundle is built.
type CreateOptions struct {
	// BundlePath is the path to the base CodeQL bundle archive (.tar.gz).
	BundlePath string
	// WorkspaceDir is the CodeQL workspace containing the packs to add.
	WorkspaceDir string
	// Packs is the list of pack names to include (e.g. "foo/cpp-customizations").
	Packs []string
	// OutputPath is where the resulting bundle archive is written.
	// If Platforms is non-empty, this must be a directory; otherwise it is a file path.
	OutputPath string
	// Platforms restricts output to specific platforms ("linux64", "osx64", "win64").
	// Empty means a single platform-agnostic bundle.
	Platforms []string
	// NoPrecompile skips pre-compilation when bundling packs.
	NoPrecompile bool
}

// Create builds a custom CodeQL bundle by extending the base bundle with
// additional packs from the workspace.
func Create(opts CreateOptions) error {
	// 1. Extract the base bundle into a temp directory.
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

	// 2. Locate the codeql binary inside the extracted bundle.
	codeqlBin := filepath.Join(bundleDir, "codeql")
	if runtime.GOOS == "windows" {
		codeqlBin = filepath.Join(bundleDir, "codeql.exe")
	}
	if _, err := os.Stat(codeqlBin); err != nil {
		return fmt.Errorf("codeql binary not found in bundle at %s: %w", codeqlBin, err)
	}
	slog.Debug("Using bundle codeql binary", "path", codeqlBin)

	// 3. Query bundle metadata: version and languages.
	version, err := codeqlVersion(codeqlBin)
	if err != nil {
		return fmt.Errorf("querying bundle codeql version: %w", err)
	}
	slog.Info("Bundle codeql version", "version", version)

	qlxSupported := supportsQlx(version)
	slog.Debug("QLX support", "enabled", qlxSupported)

	languages, err := resolveLanguages(codeqlBin)
	if err != nil {
		return fmt.Errorf("resolving languages: %w", err)
	}
	slog.Debug("Bundle languages", "languages", languages)

	// 4. List packs in the bundle and the workspace.
	slog.Info("Listing bundle packs", "dir", bundleDir)
	bundlePacks, err := ListPacks(codeqlBin, bundleDir)
	if err != nil {
		return fmt.Errorf("listing bundle packs: %w", err)
	}

	slog.Info("Listing workspace packs", "dir", opts.WorkspaceDir)
	allWorkspacePacks, err := ListPacks(codeqlBin, opts.WorkspaceDir)
	if err != nil {
		return fmt.Errorf("listing workspace packs: %w", err)
	}

	// Filter out test packs (packs with an extractor field).
	var workspacePacks []*Pack
	for _, p := range allWorkspacePacks {
		if p.IsTestPack() {
			slog.Debug("Skipping test pack", "pack", p.Config.Name)
			continue
		}
		workspacePacks = append(workspacePacks, p)
	}

	// Validate: all packs must have a scope.
	for _, wp := range workspacePacks {
		if wp.Config.Scope() == "" {
			return fmt.Errorf("pack %q has no scope; all bundled packs must be scoped", wp.Config.Name)
		}
	}

	// Filter to the requested packs.
	selected, err := selectPacks(workspacePacks, opts.Packs)
	if err != nil {
		return err
	}

	// 5. Resolve dependencies.
	allPacks := append(bundlePacks, workspacePacks...)
	if err := ResolveDeps(workspacePacks, allPacks); err != nil {
		return fmt.Errorf("resolving dependencies: %w", err)
	}

	// 6. Build the processing order.
	order, stdlibCustomizations, err := BuildProcessingOrder(selected, bundlePacks)
	if err != nil {
		return fmt.Errorf("building processing order: %w", err)
	}
	slog.Info("Processing packs", "count", len(order))

	// 7. Process each pack in topological order.
	qlpacksDir := filepath.Join(bundleDir, "qlpacks")
	ctx := &buildCtx{
		codeqlBin:            codeqlBin,
		bundleDir:            bundleDir,
		qlpacksDir:           qlpacksDir,
		tmpDir:               tmpDir,
		stdlibCustomizations: stdlibCustomizations,
		noPrecompile:         opts.NoPrecompile,
		qlxSupported:         qlxSupported,
	}

	for _, pack := range order {
		if err := ctx.processPack(pack); err != nil {
			return fmt.Errorf("processing %s: %w", pack.Config.Name, err)
		}
	}

	// 8. Archive the modified bundle.
	if len(opts.Platforms) == 0 {
		// Platform-agnostic single archive.
		outputPath := opts.OutputPath
		if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
			outputPath = filepath.Join(outputPath, "codeql-bundle.tar.gz")
		}
		slog.Info("Creating platform-agnostic bundle", "output", outputPath)
		if err := CreateTarGz(outputPath, bundleDir, nil); err != nil {
			return fmt.Errorf("creating bundle archive: %w", err)
		}
	} else {
		// One archive per platform.
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

// buildCtx holds shared state during pack processing.
type buildCtx struct {
	codeqlBin            string
	bundleDir            string
	qlpacksDir           string
	tmpDir               string
	stdlibCustomizations map[*Pack][]*Pack
	noPrecompile         bool
	qlxSupported         bool
}

func (c *buildCtx) processPack(p *Pack) error {
	switch {
	case p.Kind == CustomizationPack:
		return c.bundleCustomizationPack(p)
	case p.Kind == LibraryPack && p.Config.Scope() == "codeql":
		return c.bundleStdlibPack(p)
	case p.Kind == LibraryPack:
		return c.bundleLibraryPack(p)
	case p.Kind == QueryPack && p.Config.Scope() == "codeql":
		return c.recreateStdlibQueryPack(p)
	default:
		return c.recreateQueryPack(p)
	}
}

// bundleCustomizationPack strips the stdlib dependency (to avoid circular
// deps) and runs `codeql pack bundle` into <bundleDir>/qlpacks.
func (c *buildCtx) bundleCustomizationPack(p *Pack) error {
	slog.Info("Bundling customization pack", "pack", p.Config.Name)

	packCopy, err := c.copyPack(p)
	if err != nil {
		return err
	}

	// Remove the stdlib dependency to prevent circular dependency.
	slog.Debug("Removing stdlib dependency to prevent circular dep", "pack", packCopy.Config.Name)
	if err := saveConfig(packCopy.YmlPath, map[string]any{"dependencies": map[string]string{}}); err != nil {
		return fmt.Errorf("clearing dependencies in copy: %w", err)
	}

	return c.packBundle(packCopy)
}

// bundleStdlibPack adds customization pack dependencies to the stdlib lib pack,
// ensures a Customizations.qll exists, updates it with imports, removes the
// original from the bundle, and re-bundles.
func (c *buildCtx) bundleStdlibPack(p *Pack) error {
	slog.Info("Bundling stdlib library pack", "pack", p.Config.Name)

	packCopy, err := c.copyPack(p)
	if err != nil {
		return err
	}

	// Add customization packs as dependencies.
	customizationPacks := c.stdlibCustomizations[p]
	newDeps := make(map[string]string)
	for _, cp := range customizationPacks {
		newDeps[cp.Config.Name] = cp.Config.Version
		slog.Debug("Adding customization dep to stdlib", "pack", packCopy.Config.Name, "dep", cp.Config.Name)
	}
	if err := saveConfig(packCopy.YmlPath, map[string]any{"dependencies": newDeps}); err != nil {
		return fmt.Errorf("updating dependencies: %w", err)
	}

	// Ensure Customizations.qll exists.
	if !packCopy.IsCustomizable() {
		if err := c.addCustomizationSupport(packCopy); err != nil {
			return fmt.Errorf("adding customization support: %w", err)
		}
	}

	// Append imports of customization packs to Customizations.qll.
	f, err := os.OpenFile(packCopy.CustomizationsPath(), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening Customizations.qll: %w", err)
	}
	for _, cp := range customizationPacks {
		line := fmt.Sprintf("import %s.Customizations\n", cp.Config.ModuleName())
		slog.Debug("Appending import to Customizations.qll", "import", strings.TrimSpace(line))
		if _, err := f.WriteString(line); err != nil {
			f.Close()
			return err
		}
	}
	f.Close()

	// Remove the original stdlib lib pack from the bundle (scope/name/ directory).
	originalDir := filepath.Dir(filepath.Dir(p.YmlPath)) // .../qlpacks/codeql/cpp-all
	slog.Debug("Removing original stdlib pack", "dir", originalDir)
	if err := os.RemoveAll(originalDir); err != nil {
		return fmt.Errorf("removing original stdlib pack: %w", err)
	}

	return c.packBundle(packCopy)
}

// bundleLibraryPack bundles a non-stdlib library pack into the bundle.
func (c *buildCtx) bundleLibraryPack(p *Pack) error {
	slog.Info("Bundling library pack", "pack", p.Config.Name)
	packCopy, err := c.copyPack(p)
	if err != nil {
		return err
	}
	return c.packBundle(packCopy)
}

// recreateStdlibQueryPack cleans up the existing compiled artifacts and
// recreates the query pack using the modified bundle as its dependency source.
func (c *buildCtx) recreateStdlibQueryPack(p *Pack) error {
	slog.Info("Recreating stdlib query pack", "pack", p.Config.Name)

	packCopy, err := c.copyPack(p)
	if err != nil {
		return err
	}

	// Remove lock file, installed deps, cache, and qlx files.
	_ = os.Remove(packCopy.LockFilePath())
	_ = os.RemoveAll(packCopy.DepsPath())
	_ = os.RemoveAll(packCopy.CachePath())

	if c.qlxSupported {
		if err := removeGlob(packCopy.Dir(), "**/*.qlx"); err != nil {
			slog.Warn("Error removing qlx files", "error", err)
		}
	}

	// Remove the original query pack from the bundle.
	originalDir := filepath.Dir(filepath.Dir(p.YmlPath))
	slog.Debug("Removing original stdlib query pack", "dir", originalDir)
	if err := os.RemoveAll(originalDir); err != nil {
		return fmt.Errorf("removing original stdlib query pack: %w", err)
	}

	return c.packCreate(packCopy, c.bundleDir)
}

// recreateQueryPack rewrites dependencies and creates a non-stdlib query pack.
func (c *buildCtx) recreateQueryPack(p *Pack) error {
	slog.Info("Creating query pack", "pack", p.Config.Name)

	packCopy, err := c.copyPack(p)
	if err != nil {
		return err
	}

	// Rewrite dependencies to resolved versions.
	newDeps := make(map[string]string, len(packCopy.Deps))
	for _, dep := range packCopy.Deps {
		newDeps[dep.Config.Name] = dep.Config.Version
	}
	if err := saveConfig(packCopy.YmlPath, map[string]any{"dependencies": newDeps}); err != nil {
		return fmt.Errorf("rewriting dependencies: %w", err)
	}

	return c.packCreate(packCopy, c.bundleDir)
}

// packBundle runs `codeql pack bundle --pack-path=<qlpacksDir> -- <packDir>`.
func (c *buildCtx) packBundle(p *Pack) error {
	args := []string{"pack", "bundle", "--format=json", fmt.Sprintf("--pack-path=%s", c.qlpacksDir)}
	if c.noPrecompile {
		args = append(args, "--no-precompile")
	}
	args = append(args, "--", p.Dir())
	slog.Debug("Running codeql pack bundle", "pack", p.Config.Name, "args", args)
	if _, err := executil.NewRunner(c.codeqlBin).Run(args...); err != nil {
		return fmt.Errorf("codeql pack bundle %s: %w", p.Config.Name, err)
	}
	return nil
}

// packCreate runs `codeql pack create --output=<qlpacksDir> -- <packDir>`.
// additionalPacks is an optional list of extra pack search paths.
func (c *buildCtx) packCreate(p *Pack, additionalPacks ...string) error {
	args := []string{
		"pack", "create",
		"--format=json",
		fmt.Sprintf("--output=%s", c.qlpacksDir),
		"--threads=0",
		"--no-default-compilation-cache",
	}
	if c.noPrecompile {
		args = append(args, "--no-precompile")
	}
	if c.qlxSupported {
		args = append(args, "--qlx")
	}
	if len(additionalPacks) > 0 {
		args = append(args, fmt.Sprintf("--additional-packs=%s", strings.Join(additionalPacks, string(os.PathListSeparator))))
	}
	args = append(args, "--", p.Dir())
	slog.Debug("Running codeql pack create", "pack", p.Config.Name, "args", args)
	if _, err := executil.NewRunner(c.codeqlBin).Run(args...); err != nil {
		return fmt.Errorf("codeql pack create %s: %w", p.Config.Name, err)
	}
	return nil
}

// copyPack copies a pack directory to a temp subdirectory and returns a new
// Pack pointing at the copy.
func (c *buildCtx) copyPack(p *Pack) (*Pack, error) {
	scope := p.Config.Scope()
	name := p.Config.PackName()
	version := p.Config.Version
	if version == "" {
		version = "0.0.0"
	}
	destDir := filepath.Join(c.tmpDir, "work", scope, name, version)
	slog.Debug("Copying pack", "from", p.Dir(), "to", destDir)
	if err := copyDir(p.Dir(), destDir); err != nil {
		return nil, fmt.Errorf("copying pack %s: %w", p.Config.Name, err)
	}
	ymlName := filepath.Base(p.YmlPath)
	return &Pack{
		YmlPath: filepath.Join(destDir, ymlName),
		Config:  p.Config,
		Kind:    p.Kind,
		Deps:    p.Deps,
	}, nil
}

// addCustomizationSupport adds a Customizations.qll to a stdlib lib pack that
// does not have one, and inserts `import Customizations` into the language module.
func (c *buildCtx) addCustomizationSupport(p *Pack) error {
	// Derive language name: "cpp-all" → "cpp"
	lang := strings.TrimSuffix(p.Config.PackName(), "-all")
	langLib := filepath.Join(p.Dir(), lang+".qll")
	if _, err := os.Stat(langLib); err != nil {
		return fmt.Errorf("cannot find language library %s.qll for %s", lang, p.Config.Name)
	}

	// Insert `import Customizations` before the first import statement.
	if err := insertBeforeFirstImport(langLib, "import Customizations"); err != nil {
		return fmt.Errorf("patching %s.qll: %w", lang, err)
	}

	// Create a minimal Customizations.qll.
	customPath := filepath.Join(p.Dir(), "Customizations.qll")
	content := fmt.Sprintf("import %s\n", lang)
	slog.Debug("Creating Customizations.qll", "path", customPath)
	return os.WriteFile(customPath, []byte(content), 0644)
}

// ---- helpers ----

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

// codeqlVersion runs `codeql version --format=json` and returns the version string.
func codeqlVersion(codeqlBin string) (string, error) {
	res, err := executil.NewRunner(codeqlBin).Run("version", "--format=json")
	if err != nil {
		return "", err
	}
	raw := res.Stdout
	var info struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &info); err != nil {
		return "", fmt.Errorf("parsing version output: %w", err)
	}
	return info.Version, nil
}

// supportsQlx returns true if the CodeQL version is >= 2.11.4.
func supportsQlx(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return false
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])
	return major > 2 ||
		(major == 2 && minor > 11) ||
		(major == 2 && minor == 11 && patch >= 4)
}

// resolveLanguages runs `codeql resolve languages --format=json` and returns language names.
func resolveLanguages(codeqlBin string) ([]string, error) {
	res, err := executil.NewRunner(codeqlBin).Run("resolve", "languages", "--format=json")
	if err != nil {
		return nil, err
	}
	raw := res.Stdout
	var langs map[string]any
	if err := json.Unmarshal(raw, &langs); err != nil {
		return nil, fmt.Errorf("parsing languages output: %w", err)
	}
	var result []string
	for k := range langs {
		result = append(result, k)
	}
	return result, nil
}

// copyDir recursively copies src to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		linfo, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if linfo.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = copyFrom(in, out)
	return err
}

// copyFrom copies all bytes from r to w.
func copyFrom(r io.Reader, w io.Writer) (int64, error) {
	return io.Copy(w, r)
}

// removeGlob removes all files matching the glob pattern under dir.
func removeGlob(dir, pattern string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		matched, err := filepath.Match(pattern[strings.LastIndex(pattern, "/")+1:], filepath.Base(path))
		if err != nil {
			return err
		}
		if matched {
			return os.Remove(path)
		}
		return nil
	})
}

// insertBeforeFirstImport inserts `line` before the first `import` statement in a .qll file.
func insertBeforeFirstImport(filePath, line string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	insertIdx := -1
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "import ") {
			insertIdx = i
			break
		}
	}
	if insertIdx < 0 {
		return fmt.Errorf("no import statement found in %s", filePath)
	}

	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:insertIdx]...)
	newLines = append(newLines, line)
	newLines = append(newLines, lines[insertIdx:]...)

	return os.WriteFile(filePath, []byte(strings.Join(newLines, "\n")+"\n"), 0644)
}
