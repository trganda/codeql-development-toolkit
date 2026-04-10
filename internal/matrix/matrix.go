package matrix

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Entry is a single cell in a GitHub Actions CI matrix.
type Entry struct {
	OS        string `json:"os"`
	CodeQLCLI string `json:"codeql_cli"`
}

// Build constructs and marshals a GitHub Actions matrix JSON from a
// comma-separated list of OS values and a CodeQL CLI version string.
func Build(osVersions, cliVersion string) ([]byte, error) {
	var entries []Entry
	for _, os := range strings.Split(osVersions, ",") {
		os = strings.TrimSpace(os)
		if os == "" {
			continue
		}
		entries = append(entries, Entry{OS: os, CodeQLCLI: cliVersion})
	}

	out, err := json.Marshal(map[string]any{"include": entries})
	if err != nil {
		return nil, fmt.Errorf("marshal matrix: %w", err)
	}
	return out, nil
}
