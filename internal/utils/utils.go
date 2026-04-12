package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// CheckWorkspace returns an error if codeql-workspace.yml does not exist under base.
// All lifecycle phases except init call this as their first step.
func CheckWorkspace(base string) error {
	if _, err := os.Stat(filepath.Join(base, "codeql-workspace.yml")); os.IsNotExist(err) {
		return fmt.Errorf("workspace not initialized — run 'qlt lifecycle init' first")
	}
	return nil
}
