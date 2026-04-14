package bundle

import "log/slog"

// processCustomizationPack is a placeholder for customization-kind packs.
// The full bundling flow (injecting Customizations.qll imports into stdlib
// packs, topological ordering) is not yet implemented in the simplified
// pipeline; the pack is skipped with a warning.
func processCustomizationPack(_ *buildCtx, p *Pack) error {
	slog.Warn("Customization packs are not yet supported in bundle create; skipping",
		"pack", p.Config.Name)
	return nil
}

// processLibraryPack is a placeholder for library-kind packs. Library bundling
// is not yet implemented in the simplified pipeline; the pack is skipped with
// a warning.
func processLibraryPack(_ *buildCtx, p *Pack) error {
	slog.Warn("Library packs are not yet supported in bundle create; skipping",
		"pack", p.Config.Name)
	return nil
}
