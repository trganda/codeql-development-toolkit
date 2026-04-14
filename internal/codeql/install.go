package codeql

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/trganda/codeql-development-toolkit/internal/archive"
	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

const cliDownloadBase = "https://github.com/github/codeql-cli-binaries/releases/download"
const bundleDownloadBase = "https://github.com/github/codeql-action/releases/download"

var downloadClient = &http.Client{Timeout: 30 * time.Minute}

// Install downloads and installs the CodeQL CLI or bundle based on config.
// When EnableCustomCodeQLBundles is true in config, the bundle is installed;
// otherwise the standalone CLI is used. version overrides the config value for
// CLI installs.
func Install(base, platform string) error {
	cfg := config.MustLoadFromFile(base)

	slog.Info("Installing CodeQL CLI", "version", cfg.CodeQLCLI)
	return installCLI(cfg.CodeQLCLI, platform)
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

// bundlePlatformAsset returns the release asset filename for the CodeQL bundle.
// With platform "all" or the multi-arch default, returns "codeql-bundle.tar.gz".
// Otherwise returns "codeql-bundle-<platform>.tar.gz", auto-detecting the
// platform from the current OS when platform is empty.
func bundlePlatformAsset(platform string) (string, error) {
	if platform != "" {
		if platform != "all" {
			return "codeql-bundle-" + platform + ".tar.gz", nil
		}
		return "codeql-bundle.tar.gz", nil
	}

	switch runtime.GOOS {
	case "linux":
		return "codeql-bundle-linux64.tar.gz", nil
	case "darwin":
		return "codeql-bundle-osx64.tar.gz", nil
	case "windows":
		return "codeql-bundle-win64.tar.gz", nil
	default:
		return "", fmt.Errorf("unsupported platform %s; use --platform to override", runtime.GOOS)
	}
}

// installBundle downloads and unpacks the CodeQL bundle to
// $HOME/.qlt/bundle/<md5(bundleName)>. Skips the download if a local archive
// with a matching checksum already exists.
func installBundle(base, bundleName, platform string) error {
	installDir, err := paths.BundleInstallDir(bundleName)
	if err != nil {
		return fmt.Errorf("resolve bundle install directory: %w", err)
	}

	archiveName, err := bundlePlatformAsset(platform)
	if err != nil {
		return err
	}
	assetURL := fmt.Sprintf("%s/%s/%s", bundleDownloadBase, bundleName, archiveName)
	archivePath := filepath.Join(installDir, archiveName)
	codeqlDir := filepath.Join(installDir, "codeql")

	remoteDigest, err := fetchBundleRemoteChecksum(bundleName, archiveName, installDir)
	if err != nil {
		return fmt.Errorf("resolve remote bundle checksum: %w", err)
	}

	// Skip download when the cached archive already matches.
	if _, statErr := os.Stat(archivePath); statErr == nil {
		localDigest, err := localFileSHA256(archivePath)
		if err == nil && localDigest == remoteDigest {
			slog.Info("CodeQL bundle already up-to-date, skipping download", "bundle", bundleName)
			if _, statErr := os.Stat(codeqlDir); statErr == nil {
				fmt.Printf("CodeQL bundle %s already installed at %s\n", bundleName, codeqlDir)
				return nil
			}
			slog.Info("Extracting existing bundle archive", "archive", archivePath, "dest", installDir)
			if err := archive.ExtractTarGz(archivePath, installDir); err != nil {
				return fmt.Errorf("extract bundle: %w", err)
			}
			fmt.Printf("CodeQL bundle %s installed at %s\n", bundleName, codeqlDir)
			return nil
		}
	}

	fmt.Printf("Downloading CodeQL bundle %s...\n", bundleName)
	slog.Debug("Downloading bundle", "url", assetURL, "dest", archivePath)
	if err := downloadFile(assetURL, archivePath); err != nil {
		return fmt.Errorf("download bundle: %w", err)
	}

	localDigest, err := localFileSHA256(archivePath)
	if err != nil {
		return fmt.Errorf("compute bundle checksum: %w", err)
	}
	if localDigest != remoteDigest {
		_ = os.Remove(archivePath)
		return fmt.Errorf("bundle checksum mismatch: expected %s, got %s", remoteDigest, localDigest)
	}
	slog.Debug("Bundle checksum verified", "digest", localDigest)

	fmt.Printf("Extracting bundle to %s...\n", installDir)
	if err := os.RemoveAll(codeqlDir); err != nil {
		return fmt.Errorf("remove stale bundle install: %w", err)
	}
	if err := archive.ExtractTarGz(archivePath, installDir); err != nil {
		return fmt.Errorf("extract bundle: %w", err)
	}

	fmt.Printf("CodeQL bundle %s installed at %s\n", bundleName, codeqlDir)
	return nil
}

// fetchBundleRemoteChecksum downloads the .checksum.txt for the bundle archive.
func fetchBundleRemoteChecksum(bundleName, assetName, destDir string) (string, error) {
	url := fmt.Sprintf("%s/%s/%s.checksum.txt", bundleDownloadBase, bundleName, assetName)
	slog.Debug("Fetching bundle checksum file", "url", url)

	resp, err := downloadClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch bundle checksum: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bundle checksum file returned status %d (%s)", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read bundle checksum body: %w", err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("create bundle dest dir: %w", err)
	}
	checksumPath := filepath.Join(destDir, assetName+".checksum.txt")
	if err := os.WriteFile(checksumPath, body, 0644); err != nil {
		return "", fmt.Errorf("save bundle checksum file: %w", err)
	}
	slog.Debug("Saved bundle checksum file", "path", checksumPath)

	return parseChecksum(body, assetName)
}

// fetchRemoteChecksum downloads the .checksum.txt file published alongside each
// release asset, persists it to destDir, and returns the lowercase SHA-256 hex
// digest for assetName. The checksum file contains lines in standard shasum
// format:
//
//	<hex>  <filename>
func fetchRemoteChecksum(version, assetName, destDir string) (string, error) {
	tag := paths.VersionTag(version)
	url := fmt.Sprintf("%s/%s/%s.checksum.txt", cliDownloadBase, tag, assetName)
	slog.Debug("Fetching checksum file", "url", url)

	resp, err := downloadClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch checksum: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum file returned status %d (%s)", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read checksum body: %w", err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("create dest dir: %w", err)
	}
	checksumPath := filepath.Join(destDir, assetName+".checksum.txt")
	if err := os.WriteFile(checksumPath, body, 0644); err != nil {
		return "", fmt.Errorf("save checksum file: %w", err)
	}
	slog.Debug("Saved checksum file", "path", checksumPath)

	return parseChecksum(body, assetName)
}

// parseChecksum extracts the lowercase SHA-256 hex digest for assetName from
// the contents of a shasum-style checksum file.
func parseChecksum(content []byte, assetName string) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Format: "<hash>  <filename>" or "<hash> <filename>"
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.EqualFold(fields[1], assetName) {
			return strings.ToLower(fields[0]), nil
		}
		// Single-field file (just the hash)
		if len(fields) == 1 {
			return strings.ToLower(fields[0]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read checksum file: %w", err)
	}
	return "", fmt.Errorf("checksum for %s not found in checksum file", assetName)
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

// localFileSHA256 computes the SHA-256 hex digest of the file at path.
func localFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
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
		slog.Info("CodeQL CLI %s already installed at %s", version, codeqlDir)
		return nil
	}

	slog.Info("Extracting existing archive", "zip", zipPath, "dest", installDir)
	if err := os.RemoveAll(codeqlDir); err != nil {
		return fmt.Errorf("remove stale install: %w", err)
	}
	if err := archive.ExtractZip(zipPath, installDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	slog.Info("CodeQL CLI %s installed at %s", version, codeqlDir)
	return nil
}
