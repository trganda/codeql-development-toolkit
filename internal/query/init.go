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
func InitWorkspace(base, scope, codeqlVersion, bundleVersion string, useBundle, overwriteExisting bool) (*config.QLTConfig, error) {
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
	slog.Info("Initialized CodeQL workspace", "path", dst)

	cfg, err := config.LoadFromFile(base)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if cfg == nil {
		cfg = &config.QLTConfig{}
	}

	if useBundle {
		cfg.EnableCustomCodeQLBundles = true
	}
	if scope != "" {
		cfg.Scope = scope
	}
	// Only overwrite CLI and bundle versions in config if they are not already set, or if --overwrite-existing is provided.
	if cfg.CodeQLCLI == "" || overwriteExisting {
		cfg.CodeQLCLI = codeqlVersion
	}
	if err := cfg.SaveToFile(base); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}
	if useBundle {
		slog.Info("Enabled custom CodeQL bundles in config")
	}
	if scope != "" {
		slog.Info("Saved scope to config", "scope", scope)
	} else {
		slog.Warn("Scope was not specificed")
	}

	return cfg, nil
}
