package bundle

import "fmt"

// CreateOptions controls how the custom bundle is built.
type CreateOptions struct {
	// BundlePath is the path to the base CodeQL bundle archive (.tar.gz).
	BundlePath string
	// WorkspaceDir is the CodeQL workspace containing the packs to add.
	WorkspaceDir string
	// Packs is the list of pack names to include (e.g. "foo/cpp-customizations").
	Packs []string
	// OutputPath is where the resulting bundle archive is written.
	// If Platforms is non-empty, this is treated as a directory; otherwise as a file path.
	OutputPath string
	// Platforms restricts output to specific platforms ("linux64", "osx64", "win64").
	// Empty means a single platform-agnostic bundle.
	Platforms []string
	// NoPrecompile skips pre-compilation when bundling packs.
	NoPrecompile bool
	// Minimal creates a minimal bundle with only the selected packs and no additional
	// dependencies. Currently a no-op; reserved for future use.
	Minimal bool
}

// Validate checks that required fields are set and values are well-formed.
func (o *CreateOptions) Validate() error {
	if o.BundlePath == "" {
		return fmt.Errorf("BundlePath is required")
	}
	if o.WorkspaceDir == "" {
		return fmt.Errorf("WorkspaceDir is required")
	}
	if o.OutputPath == "" {
		return fmt.Errorf("OutputPath is required")
	}
	return ValidatePlatforms(o.Platforms)
}
