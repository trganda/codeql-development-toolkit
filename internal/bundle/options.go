package bundle

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
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

func NewCreateOptions(base, bundlePath string, noPrecompile, minimal bool, platforms []string) (*CreateOptions, error) {
	var (
		output string
		packs  []string
		err    error
	)

	cfg := config.MustLoadFromFile(base)
	for _, p := range cfg.CodeQLPackConfiguration {
		if p.Bundle {
			slog.Debug("Pack configured for bundling", "pack", p.Name)
			packs = append(packs, p.Name)
		}
	}

	if len(packs) == 0 {
		return nil, fmt.Errorf("no packs configured for bundling in qlt.conf.json; set Bundle=true on at least one CodeQLPackConfiguration entry")
	}

	if bundlePath == "" {
		if cfg.CodeQLCLIBundle == "" {
			return nil, fmt.Errorf("CodeQLCLIBundle is not set in qlt.conf.json; run 'qlt codeql set version' or provide --bundle")
		}
		bundlePath, err = paths.BundleArchivePath(cfg.CodeQLCLIBundle)
		if err != nil {
			return nil, fmt.Errorf("resolving bundle path: %w", err)
		}
	}

	output, err = paths.CustomBundlePath(base, cfg.CodeQLCLIBundle)
	if err != nil {
		return nil, fmt.Errorf("resolving custom bundle output path: %w", err)
	}
	err = os.MkdirAll(output, 0755)
	if err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	if err := ValidatePlatforms(platforms); err != nil {
		return nil, err
	}

	return &CreateOptions{
		BundlePath:   bundlePath,
		OutputPath:   output,
		WorkspaceDir: base,
		NoPrecompile: noPrecompile,
		Minimal:      minimal,
		Platforms:    platforms,
		Packs:        packs,
	}, nil
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
