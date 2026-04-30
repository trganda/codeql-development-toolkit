package bundle

import "strings"

// makePlatformFilter returns a filter function for archive.CreateTarGz that
// excludes paths not belonging to the target platform.
func makePlatformFilter(platform string, languages []string) func(string) bool {
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

// platformExclusions returns the set of slash-prefixed path prefixes that
// should be excluded for the given target platform. Each entry is relative to
// the bundle root (no leading slash).
//
// platform is one of "linux64", "osx64", "win64".
// languages is the set of language names returned by `codeql resolve languages`.
func platformExclusions(platform string, languages []string) []string {
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
