package codeql

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/trganda/codeql-development-toolkit/internal/archive"
	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

const cliDownloadBase = "https://github.com/github/codeql-cli-binaries/releases/download"

// const bundleDownloadBase = "https://github.com/github/codeql-action/releases/download"

var downloadClient = &http.Client{Timeout: 30 * time.Minute}

// Install downloads and installs the CodeQL CLI or bundle based on config.
// When EnableCustomCodeQLBundles is true in config, the bundle is installed;
// otherwise the standalone CLI is used. version overrides the config value for
// CLI installs.
func Install(base, platform string) error {
	cfg := config.MustLoadFromFile(base)

	slog.Info("Installing CodeQL CLI", "version", cfg.CodeQLCLIVersion)
	return installCLI(cfg.CodeQLCLIVersion, platform)
}

func Download(version, platform string) (p string, err error) {
	asset, err := platformAsset(platform)
	if err != nil {
		return "", err
	}

	installDir, err := paths.CLIInstallDir(version)
	if err != nil {
		return "", fmt.Errorf("resolve install directory: %w", err)
	}

	tag := paths.VersionTag(version)
	assetURL := fmt.Sprintf("%s/%s/%s", cliDownloadBase, tag, asset)
	zipPath := filepath.Join(installDir, asset)

	// Skip downlaod when the cached zip already matches.
	if _, statErr := os.Stat(zipPath); statErr == nil {
		return zipPath, nil
	}

	slog.Info("Downloading CodeQL CLI", "version", version)
	if err := downloadFile(assetURL, zipPath); err != nil {
		return "", fmt.Errorf("download: %w", err)
	}

	return zipPath, nil
}

// platformAsset returns the release asset filename for the current OS/arch,
// or the override value when platform is non-empty (e.g. "linux64", "osx-arm64").
func platformAsset(platform string) (string, error) {
	if platform != "" {
		if platform == "all" {
			return "codeql.zip", nil
		}
		return "codeql-" + platform + ".zip", nil
	}
	switch runtime.GOOS {
	case "linux":
		return "codeql-linux64.zip", nil
	case "darwin":
		return "codeql-osx64.zip", nil
	case "windows":
		return "codeql-win64.zip", nil
	default:
		return "", fmt.Errorf("unsupported platform %s; use --platform to override", runtime.GOOS)
	}
}

// downloadFile streams url to dst, writing via a temp file and doing an atomic rename.
func downloadFile(url, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmp) }()

	resp, err := downloadClient.Get(url)
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_ = f.Close()
		return fmt.Errorf("download %s: unexpected status %d", url, resp.StatusCode)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		return fmt.Errorf("write download: %w", err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

// installCLI downloads and unpacks the CodeQL CLI to
// $HOME/.qlt/packages/<md5(version)>. Skips the download if a local zip with a
// matching checksum already exists.
func installCLI(version, platform string) error {

	zipPath, err := Download(version, platform)
	if err != nil {
		return err
	}

	installDir, err := paths.CLIInstallDir(version)
	if err != nil {
		return fmt.Errorf("resolve install dir: %w", err)
	}
	codeqlDir := filepath.Join(installDir, "codeql")
	// Skip extracting when the target directory already exists.
	if _, statErr := os.Stat(codeqlDir); statErr == nil {
		slog.Info("CodeQL CLI already installed at ", "version", version, "path", codeqlDir)
		return nil
	}

	slog.Info("Extracting existing archive", "zip", zipPath, "dest", installDir)
	if err := os.RemoveAll(codeqlDir); err != nil {
		return fmt.Errorf("remove stale install: %w", err)
	}
	if err := archive.ExtractZip(zipPath, installDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	slog.Info("CodeQL CLI installed", "version", version, "path", codeqlDir)
	return nil
}
