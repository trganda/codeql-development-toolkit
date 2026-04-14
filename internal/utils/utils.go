package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// CheckWorkspace returns an error if qlt.conf.json or codeql-workspace.yml does not exist under base.
// All lifecycle phases except init call this as their first step.
func CheckWorkspace(base string) error {
	if _, err := os.Stat(filepath.Join(base, "qlt.conf.json")); os.IsNotExist(err) {
		return fmt.Errorf("No qlt.config.json found in workspace — run 'qlt lifecycle init' first")
	}

	if _, err := os.Stat(filepath.Join(base, "codeql-workspace.yml")); os.IsNotExist(err) {
		return fmt.Errorf("No codeql-workspace.yml found in workspace — run 'qlt lifecycle init' first")
	}
	return nil
}
