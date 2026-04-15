package bundle

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/pack"
)

// classify determines the kind of a pack.
// A pack is a CustomizationPack if it is a library pack AND has a
// <name_underscored>/Customizations.qll file relative to its directory.
// e.g. "foo/cpp-customizations" → check for "foo/cpp_customizations/Customizations.qll"
func classify(p *pack.Pack) pack.PackKind {
	kind := pack.QueryPack
	if p.Config.Library {
		normalized := strings.ReplaceAll(p.Config.Name, "-", "_")
		customPath := filepath.Join(p.Dir(), normalized, "Customizations.qll")
		if _, err := os.Stat(customPath); err == nil {
			kind = pack.CustomizationPack
		} else {
			kind = pack.LibraryPack
		}
	}
	return kind
}

// PackProcessor encapsulates the build-time handling of a single workspace
// pack while assembling a custom CodeQL bundle. Each pack kind has its own
// implementation, selected once at pack-discovery time so the orchestrator
// loop in CustomBundle.Create can call Process without further dispatch.
type PackProcessor interface {
	Process(p *pack.Pack) error
}

// newPackProcessor returns the PackProcessor matching kind, bound to cb.
// It is the only place in the package that branches on PackKind; once a
// processor is created, behaviour is dispatched through the interface.
func newPackProcessor(cb *CustomBundle, kind pack.PackKind) PackProcessor {
	switch kind {
	case pack.CustomizationPack:
		return &customizationPack{cb: cb}
	case pack.LibraryPack:
		return &libraryPack{cb: cb}
	default:
		return &queryPack{cb: cb}
	}
}

// queryPack runs the simplified 6-step bundling flow for a query
// pack: copy → pack install → pack create. Resolved dependencies are folded
// into the bundle by CustomBundle.Create after every processor has run.
type queryPack struct {
	cb *CustomBundle
}

func (q *queryPack) Process(p *pack.Pack) error {
	slog.Info("Processing query pack", "pack", p.Config.Name)

	packCopy, err := p.CopyTo(filepath.Join(q.cb.tmpDir, "temp"))
	if err != nil {
		return err
	}

	// Remove the .cache and .codeql folder from the copied pack if they exist, to ensure a clean install.
	os.RemoveAll(packCopy.DepsPath())
	os.RemoveAll(packCopy.CachePath())
	slog.Debug("Removed .cache and .codeql directories", "pack", p.Config.Name)

	if _, err := q.cb.tmpCodeQLCLI.PackInstall(packCopy.Dir(), q.cb.commonCachesDir); err != nil {
		return fmt.Errorf("codeql pack install %s: %w", p.Config.Name, err)
	}

	if _, err := q.cb.tmpCodeQLCLI.PackCreate(packCopy.Dir(), q.cb.tmpQlPacksDir, q.cb.commonCachesDir); err != nil {
		return fmt.Errorf("codeql pack create %s: %w", p.Config.Name, err)
	}

	return nil
}

// customizationPack is a placeholder for customization-kind packs.
// Injecting Customizations.qll imports into stdlib packs and topologically
// re-bundling is not yet implemented; the pack is skipped with a warning.
type customizationPack struct {
	cb *CustomBundle
}

func (c *customizationPack) Process(p *pack.Pack) error {
	slog.Warn("Customization packs are not yet supported in bundle create; skipping",
		"pack", p.Config.Name)
	return nil
}

// libraryPack is a placeholder for library-kind packs. Library
// bundling is not yet implemented; the pack is skipped with a warning.
type libraryPack struct {
	cb *CustomBundle
}

func (l *libraryPack) Process(p *pack.Pack) error {
	slog.Warn("Library packs are not yet supported in bundle create; skipping",
		"pack", p.Config.Name)

	packCopy, err := p.CopyTo(filepath.Join(l.cb.tmpDir, "temp"))
	if err != nil {
		return err
	}

	if _, err := l.cb.tmpCodeQLCLI.PackBundle(packCopy.Dir(), l.cb.tmpQlPacksDir, l.cb.commonCachesDir); err != nil {
		return fmt.Errorf("codeql pack bundle %s: %w", p.Config.Name, err)
	}

	return nil
}
