package query

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/trganda/codeql-development-toolkit/internal/config"
	tmpl "github.com/trganda/codeql-development-toolkit/internal/template"
)

// InitWorkspace initializes a CodeQL query workspace under base.
// It writes codeql-workspace.yml and optionally updates qlt.conf.json
// when useBundle or scope is set.
func InitWorkspace(base, scope, codeqlVersion string, overwriteExisting bool) (*config.QLTConfig, error) {
	if err := os.MkdirAll(base, 0755); err != nil {
		return nil, fmt.Errorf("create base directory: %w", err)
	}

	tmplContent, err := tmpl.Get("query/codeql-workspace.tmpl")
	if err != nil {
		return nil, fmt.Errorf("load workspace template: %w", err)
	}

	dst := filepath.Join(base, "codeql-workspace.yml")
	if err := tmpl.WriteFile(tmplContent, dst, nil, overwriteExisting); err != nil {
		return nil, err
	}
	slog.Info("Initialized CodeQL workspace configuration file", "path", dst)

	cfg, err := config.LoadFromFile(base)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Preserve existing config unless overwrite is requested.
	if cfg != nil && !overwriteExisting {
		slog.Debug("Existing config loaded", "path", config.ConfigFilePath(base))
		return cfg, nil
	}
	if cfg == nil {
		cfg = &config.QLTConfig{}
	}

	cfg.CodeQLCLI = codeqlVersion
	if scope != "" {
		cfg.Scope = scope
		slog.Info("Saved scope to config", "scope", scope)
	} else {
		slog.Warn("Scope was not specified")
	}

	if err := cfg.SaveToFile(base); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}
	return cfg, nil
}
