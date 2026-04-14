package bundle

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// CollectConfiguredPacks returns the names of packs with Bundle=true in the
// qlt.conf.json. Returns an error if none are configured.
func CollectConfiguredPacks(cfg *config.QLTConfig) ([]string, error) {
	var packs []string
	for _, p := range cfg.CodeQLPackConfiguration {
		if p.Bundle {
			packs = append(packs, p.Name)
		}
	}
	if len(packs) == 0 {
		return nil, fmt.Errorf("no packs configured for bundling in qlt.conf.json; set Bundle=true on at least one CodeQLPackConfiguration entry")
	}
	return packs, nil
}

// ResolveBundleArchive returns the base bundle archive path. If override is
// non-empty it is returned after existence check; otherwise the path is
// derived from cfg.CodeQLCLIBundle via paths.BundleArchivePath.
func ResolveBundleArchive(cfg *config.QLTConfig, override string) (string, error) {
	bundlePath := override
	if bundlePath == "" {
		if cfg.CodeQLCLIBundle == "" {
			return "", fmt.Errorf("CodeQLCLIBundle is not set in qlt.conf.json; run 'qlt codeql set version' or provide --bundle")
		}
		p, err := paths.BundleArchivePath(cfg.CodeQLCLIBundle)
		if err != nil {
			return "", fmt.Errorf("resolving bundle path: %w", err)
		}
		bundlePath = p
	}
	if _, err := os.Stat(bundlePath); err != nil {
		return "", fmt.Errorf("bundle archive not found at %s: %w", bundlePath, err)
	}
	return bundlePath, nil
}

// ResolveOutputPath returns the output archive path. If override is non-empty
// it is used verbatim; otherwise the default under base/target/custom-bundle is
// derived from cfg.CodeQLCLIBundle. The parent directory is created.
func ResolveOutputPath(base string, cfg *config.QLTConfig, override string) (string, error) {
	output := override
	if output == "" {
		if cfg.CodeQLCLIBundle == "" {
			return "", fmt.Errorf("CodeQLCLIBundle is not set in qlt.conf.json; run 'qlt codeql set version' first")
		}
		p, err := paths.CustomBundlePath(base, cfg.CodeQLCLIBundle)
		if err != nil {
			return "", fmt.Errorf("resolving custom bundle output path: %w", err)
		}
		output = p
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}
	return output, nil
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
