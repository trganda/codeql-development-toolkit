package bundle

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/executil"
	"github.com/trganda/codeql-development-toolkit/internal/pack"
)

// ListPackWithProcess runs `codeql pack ls --format=json <dir>` and returns all packs found.
func ListPackWithProcess(codeqlBin, dir string) ([]PackProcessor, error) {
	packs, err := pack.ListPacks(codeqlBin, dir)
	if err != nil {
		return nil, err
	}
	var processors []PackProcessor
	for _, p := range packs {
		processors = append(processors, newPackProcessor(p, classify(p.YmlPath, p.Config)))
	}
	return processors, nil
}

// classify determines the kind of a pack.
// A pack is a CustomizationPack if it is a library pack AND has a
// <name_underscored>/Customizations.qll file relative to its directory.
// e.g. "foo/cpp-customizations" → check for "foo/cpp_customizations/Customizations.qll"
func classify(ymlPath string, cfg pack.QlpackConfig) pack.PackKind {
	if !cfg.Library {
		return pack.QueryPack
	}
	normalized := strings.ReplaceAll(cfg.Name, "-", "_")
	customPath := filepath.Join(filepath.Dir(ymlPath), normalized, "Customizations.qll")
	if _, err := os.Stat(customPath); err == nil {
		return pack.CustomizationPack
	}
	return pack.LibraryPack
}

// PackProcessor encapsulates the build-time handling of a single workspace
// pack while assembling a custom CodeQL bundle. Each pack kind has its own
// implementation, selected once at pack-discovery time so the orchestrator
// loop in CustomBundle.Create can call Process without further dispatch.
type PackProcessor interface {
	Process(cb *CustomBundle) error
	GetPack() *pack.Pack
	GetKind() pack.PackKind
}

// newPackProcessor returns the PackProcessor matching p.Kind. It is the only
// place in the package that branches on PackKind; once a Pack has been
// classified, behaviour is dispatched through the interface.
func newPackProcessor(p *pack.Pack, kind pack.PackKind) PackProcessor {
	switch kind {
	case pack.CustomizationPack:
		return &customizationPack{pack: p}
	case pack.LibraryPack:
		return &libraryPack{pack: p}
	default:
		return &queryPack{pack: p}
	}
}

// queryPack runs the simplified 6-step bundling flow for a query
// pack: copy → pack install → pack create. Resolved dependencies are folded
// into the bundle by CustomBundle.Create after every processor has run.
type queryPack struct {
	pack *pack.Pack
}

func (q *queryPack) Process(cb *CustomBundle) error {
	p := q.pack
	slog.Info("Processing query pack", "pack", p.Config.Name)

	packCopy, err := p.CopyTo(filepath.Join(cb.tmpDir, "temp"))
	if err != nil {
		return err
	}

	runner := executil.NewRunner(cb.tmpCodeQLBin)

	if _, err := runner.Run(
		"pack", "install",
		"--format=json",
		fmt.Sprintf("--common-caches=%s", cb.commonCachesDir),
		packCopy.Dir(),
	); err != nil {
		return fmt.Errorf("codeql pack install %s: %w", p.Config.Name, err)
	}

	if _, err := runner.Run(
		"pack", "create",
		"--format=json",
		fmt.Sprintf("--output=%s", cb.tmpQlPacksDir),
		fmt.Sprintf("--common-caches=%s", cb.commonCachesDir),
		packCopy.Dir(),
	); err != nil {
		return fmt.Errorf("codeql pack create %s: %w", p.Config.Name, err)
	}

	return nil
}

func (q *queryPack) GetPack() *pack.Pack {
	return q.pack
}

func (q *queryPack) GetKind() pack.PackKind {
	return pack.QueryPack
}

// customizationPack is a placeholder for customization-kind packs.
// Injecting Customizations.qll imports into stdlib packs and topologically
// re-bundling is not yet implemented; the pack is skipped with a warning.
type customizationPack struct {
	pack *pack.Pack
}

func (c *customizationPack) Process(_ *CustomBundle) error {
	slog.Warn("Customization packs are not yet supported in bundle create; skipping",
		"pack", c.pack.Config.Name)
	return nil
}

func (c *customizationPack) GetPack() *pack.Pack {
	return c.pack
}

func (c *customizationPack) GetKind() pack.PackKind {
	return pack.CustomizationPack
}

// libraryPack is a placeholder for library-kind packs. Library
// bundling is not yet implemented; the pack is skipped with a warning.
type libraryPack struct {
	pack *pack.Pack
}

func (l *libraryPack) Process(cb *CustomBundle) error {
	slog.Warn("Library packs are not yet supported in bundle create; skipping",
		"pack", l.pack.Config.Name)

	p := l.pack
	packCopy, err := p.CopyTo(filepath.Join(cb.tmpDir, "temp"))
	if err != nil {
		return err
	}

	runner := executil.NewRunner(cb.tmpCodeQLBin)
	if _, err := runner.Run(
		"pack",
		"bundle",
		"--format=json",
		fmt.Sprintf("--output=%s", cb.tmpQlPacksDir),
		fmt.Sprintf("--common-caches=%s", cb.commonCachesDir),
		packCopy.Dir(),
	); err != nil {
		return fmt.Errorf("codeql pack bundle %s: %w", p.Config.Name, err)
	}

	return nil
}

func (l *libraryPack) GetPack() *pack.Pack {
	return l.pack
}

func (l *libraryPack) GetKind() pack.PackKind {
	return pack.LibraryPack
}
