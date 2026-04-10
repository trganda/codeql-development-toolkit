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

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/archive"
	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

const cliDownloadBase = "https://github.com/github/codeql-cli-binaries/releases/download"
const bundleDownloadBase = "https://github.com/github/codeql-action/releases/download"

var downloadClient = &http.Client{Timeout: 30 * time.Minute}

func newInstallCmd(base *string) *cobra.Command {
	var version, platform string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Download and install the CodeQL CLI binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing codeql install command",
				"base", *base, "version", version, "platform", platform)
			return runInstall(*base, version, platform)
		},
	}
	cmd.Flags().StringVar(&version, "version", "", "CodeQL CLI version to install (e.g. 2.25.1); reads qlt.conf.json when omitted")
	cmd.Flags().StringVar(&platform, "platform", "", "Platform override (e.g. linux64, osx64, win64, all); auto-detected when empty. Use 'all' to download the multi-arch bundle.")
	return cmd
}

func runInstall(base, version, platform string) error {
	cfg, err := config.LoadFromFile(base)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg != nil && cfg.EnableCustomCodeQLBundles {
		bundleName := cfg.CodeQLCLIBundle
		if bundleName == "" {
			return fmt.Errorf("EnableCustomCodeQLBundles is true but CodeQLCLIBundle is not set in qlt.conf.json\n" +
				"hint: run `qlt codeql set version` first")
		}
		slog.Info("Installing CodeQL bundle (EnableCustomCodeQLBundles=true)", "bundle", bundleName)
		return installBundle(base, bundleName, platform)
	}

	if version == "" {
		if cfg != nil && cfg.CodeQLCLI != "" {
			version = cfg.CodeQLCLI
		} else {
			return fmt.Errorf("no version specified and qlt.conf.json is missing or has no CodeQLCLI field\n" +
				"hint: run `qlt codeql set version` first, or pass --version")
		}
	}
	slog.Info("Installing CodeQL CLI", "version", version)
	return installCLI(base, version, platform)
}

// bundlePlatformAsset returns the release asset filename for the CodeQL bundle.
// With platform "all" or the multi-arch default, returns "codeql-bundle.tar.gz".
// Otherwise returns "codeql-bundle-<platform>.tar.gz", auto-detecting the
// platform from the current OS when platform is empty.
func bundlePlatformAsset(platform string) (string, error) {
	if platform != "" {
		if platform != "all" {
			return "codeql-bundle-" + platform + ".tar.gz", nil
		} else {
			return "codeql-bundle.tar.gz", nil
		}
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
	saveBundleInstallDigest(base, bundleName, remoteDigest)
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

// saveBundleInstallDigest persists the installed bundle name and its digest to
// <base>/qlt.conf.json.
func saveBundleInstallDigest(base, bundleName, digest string) {
	cfg, err := config.LoadFromFile(base)
	if err != nil || cfg == nil {
		cfg = &config.QLTConfig{}
	}
	cfg.CodeQLCLIBundle = bundleName
	cfg.CodeQLCLIDigest = digest
	if err := cfg.SaveToFile(base); err != nil {
		slog.Info("Warning: could not save bundle install digest to config", "error", err)
	}
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
func installCLI(base, version, platform string) error {
	asset, err := platformAsset(platform)
	if err != nil {
		return err
	}

	installDir, err := paths.CLIInstallDir(version)
	if err != nil {
		return fmt.Errorf("resolve install directory: %w", err)
	}

	tag := paths.VersionTag(version)
	assetURL := fmt.Sprintf("%s/%s/%s", cliDownloadBase, tag, asset)
	zipPath := filepath.Join(installDir, asset)
	codeqlDir := filepath.Join(installDir, "codeql")

	remoteDigest, err := fetchRemoteChecksum(version, asset, installDir)
	if err != nil {
		return fmt.Errorf("resolve remote checksum: %w", err)
	}

	// Skip download when the cached zip already matches.
	if _, statErr := os.Stat(zipPath); statErr == nil {
		localDigest, err := localFileSHA256(zipPath)
		if err == nil && localDigest == remoteDigest {
			slog.Info("CodeQL CLI already up-to-date, skipping download", "version", version, "asset", asset)
			if _, statErr := os.Stat(codeqlDir); statErr == nil {
				fmt.Printf("CodeQL CLI %s already installed at %s\n", version, codeqlDir)
				return nil
			}
			slog.Info("Extracting existing archive", "zip", zipPath, "dest", installDir)
			if err := archive.ExtractZip(zipPath, installDir); err != nil {
				return fmt.Errorf("extract: %w", err)
			}
			fmt.Printf("CodeQL CLI %s installed at %s\n", version, codeqlDir)
			return nil
		}
	}

	fmt.Printf("Downloading CodeQL CLI %s (%s)...\n", version, asset)
	slog.Debug("Downloading", "url", assetURL, "dest", zipPath)
	if err := downloadFile(assetURL, zipPath); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	localDigest, err := localFileSHA256(zipPath)
	if err != nil {
		return fmt.Errorf("compute checksum: %w", err)
	}
	if localDigest != remoteDigest {
		_ = os.Remove(zipPath)
		return fmt.Errorf("checksum mismatch: expected %s, got %s", remoteDigest, localDigest)
	}
	slog.Debug("Checksum verified", "digest", localDigest)

	fmt.Printf("Extracting to %s...\n", installDir)
	if err := os.RemoveAll(codeqlDir); err != nil {
		return fmt.Errorf("remove stale install: %w", err)
	}
	if err := archive.ExtractZip(zipPath, installDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	fmt.Printf("CodeQL CLI %s installed at %s\n", version, codeqlDir)
	saveInstallDigest(base, version, remoteDigest)
	return nil
}

// saveInstallDigest persists the installed version and its digest to
// <base>/qlt.conf.json so subsequent runs can verify without re-fetching.
func saveInstallDigest(base, version, digest string) {
	cfg, err := config.LoadFromFile(base)
	if err != nil || cfg == nil {
		cfg = &config.QLTConfig{}
	}
	cfg.CodeQLCLI = version
	cfg.CodeQLCLIDigest = digest
	if err := cfg.SaveToFile(base); err != nil {
		slog.Info("Warning: could not save install digest to config", "error", err)
	}
}
