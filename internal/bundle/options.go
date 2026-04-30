package bundle

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/pack"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// CreateOptions controls how the custom bundle is built.
type CreateOptions struct {
	// BundlePath is the path to the base CodeQL bundle archive (.tar.gz).
	BundlePath string
	// WorkspaceDir is the CodeQL workspace containing the packs to add.
	WorkspaceDir string
	// Packs is the list of pack names to include (e.g. "foo/cpp-customizations").
	Packs []*pack.Pack
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

func NewCreateOptions(base, bundlePath, output string, noPrecompile, minimal bool, platforms []string) (*CreateOptions, error) {
	var (
		packs []string
		err   error
	)

	cfg := config.MustLoadFromFile(base)
	for _, p := range cfg.CodeQLPackConfiguration {
		if p.Bundle {
			slog.Debug("Pack configured for bundling", "pack", p.Name)
			packs = append(packs, p.Name)
		}
	}

	if len(packs) == 0 {
		return nil, fmt.Errorf("no packs configured for bundling in qlt.conf.json; set bundle=true on at least one CodeQLPackConfiguration entry")
	}

	codeqlBin, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return nil, fmt.Errorf("resolving CodeQL binary: %w", err)
	}

	cli := codeql.NewCLI(codeqlBin)
	allPacks, err := pack.ListPacks(cli, base)
	if err != nil {
		return nil, fmt.Errorf("listing packs: %w", err)
	}

	bundlePack, err := pack.SelectPacks(allPacks, packs, true)
	if err != nil {
		return nil, fmt.Errorf("selecting packs: %w", err)
	}

	if bundlePath == "" {
		if cfg.CodeQLCLIVersion == "" {
			return nil, fmt.Errorf("bundle path not provided and CodeQLCLI version not set in qlt.conf.json; set CodeQLCLI or provide --bundle")
		}
		bundlePath, err = paths.CLIArchivePath(cfg.CodeQLCLIVersion)
		if err != nil {
			return nil, fmt.Errorf("resolving bundle path: %w", err)
		}

		// Download the base CodeQL CLI bundle if not already present.
		if _, err := os.Stat(bundlePath); err != nil {
			if _, err := codeql.Download(cfg.CodeQLCLIVersion, "all"); err != nil {
				return nil, fmt.Errorf("download CodeQL CLI: %w", err)
			}
		}
	}

	if output == "" {
		output, err = paths.CustomBundlePath(base, cfg.CodeQLCLIVersion)
		if err != nil {
			return nil, fmt.Errorf("resolving custom bundle output path: %w", err)
		}
		output, err = filepath.Abs(output)
		if err != nil {
			return nil, fmt.Errorf("resolving absolute path for custom bundle output: %w", err)
		}
	}

	err = os.MkdirAll(filepath.Dir(output), 0755)
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
		Packs:        bundlePack,
	}, nil
}

// ValidatePlatforms checks that each platform is one of linux64/osx64/win64.
func ValidatePlatforms(platforms []string) error {
	for _, p := range platforms {
		switch p {
		case "linux64", "osx64", "win64":
		default:
			return fmt.Errorf("unknown platform %q; must be one of: linux64, osx64, win64", p)
		}
	}
	return nil
}
