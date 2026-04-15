package bundle

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CreateTarGz creates a .tar.gz archive from srcDir, storing entries under
// the archive root name "codeql". The filter function receives a slash-separated
// path relative to srcDir and returns false to exclude that entry (and, for
// directories, their entire subtree).
func CreateTarGz(outputPath, srcDir string, filter func(relSlash string) bool) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output archive: %w", err)
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		relSlash := filepath.ToSlash(rel)

		if filter != nil && !filter(relSlash) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Use Lstat to detect symlinks (Walk uses Lstat internally).
		linfo, err := os.Lstat(path)
		if err != nil {
			return err
		}

		var linkTarget string
		if linfo.Mode()&os.ModeSymlink != 0 {
			linkTarget, err = os.Readlink(path)
			if err != nil {
				return err
			}
		}

		hdr, err := tar.FileInfoHeader(linfo, linkTarget)
		if err != nil {
			return err
		}
		hdr.Name = "codeql/" + relSlash

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if linfo.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(tw, f)
			return err
		}
		return nil
	})
}

// platformExclusions returns the set of slash-prefixed path prefixes that
// should be excluded for the given target platform. Each entry is relative to
// the bundle root (no leading slash).
//
// platform is one of "linux64", "osx64", "win64".
// languages is the set of language names returned by `codeql resolve languages`.
func platformExclusions(platform string, languages []string) []string {
	// Tools subdirectories per platform.
	linuxSubdirs := []string{"linux64", "linux"}
	osxSubdirs := []string{"osx64", "macos"}
	winSubdirs := []string{"win64", "windows"}

	var excludeSubdirs []string
	switch platform {
	case "linux64":
		excludeSubdirs = append(osxSubdirs, winSubdirs...)
	case "osx64":
		excludeSubdirs = append(linuxSubdirs, winSubdirs...)
	case "win64":
		excludeSubdirs = append(linuxSubdirs, osxSubdirs...)
	}

	// Base tools paths to filter.
	toolsPaths := []string{"tools"}
	for _, lang := range languages {
		toolsPaths = append(toolsPaths, lang+"/tools")
	}

	var exclusions []string
	for _, base := range toolsPaths {
		for _, sub := range excludeSubdirs {
			exclusions = append(exclusions, base+"/"+sub)
		}
	}

	// Per-platform binary exclusions.
	if platform != "win64" {
		exclusions = append(exclusions, "codeql.exe")
	}
	if platform == "win64" {
		exclusions = append(exclusions, "swift/qltest", "swift/resource-dir")
	}
	if platform == "linux64" {
		exclusions = append(exclusions, "swift/qltest/osx64", "swift/resource-dir/osx64")
	}
	if platform == "osx64" {
		exclusions = append(exclusions, "swift/qltest/linux64", "swift/resource-dir/linux64")
	}

	return exclusions
}

// MakePlatformFilter returns a filter function for CreateTarGz that excludes
// paths not belonging to the target platform.
func MakePlatformFilter(platform string, languages []string) func(string) bool {
	exclusions := platformExclusions(platform, languages)
	return func(relSlash string) bool {
		for _, excl := range exclusions {
			if relSlash == excl || strings.HasPrefix(relSlash, excl+"/") {
				return false
			}
		}
		return true
	}
}
