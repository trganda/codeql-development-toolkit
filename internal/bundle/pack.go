package bundle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/executil"
	"gopkg.in/yaml.v3"
)

// PackKind classifies a CodeQL pack.
type PackKind int

const (
	QueryPack        PackKind = iota
	LibraryPack      PackKind = iota
	CustomizationPack PackKind = iota
)

// QlpackConfig holds the fields from qlpack.yml that qlt cares about.
type QlpackConfig struct {
	Name         string            `yaml:"name"`
	Version      string            `yaml:"version"`
	Library      bool              `yaml:"library"`
	Dependencies map[string]string `yaml:"dependencies"`
	Extractor    string            `yaml:"extractor"`
}

// Scope returns the scope part of the pack name (before "/").
func (c *QlpackConfig) Scope() string {
	if idx := strings.Index(c.Name, "/"); idx >= 0 {
		return c.Name[:idx]
	}
	return ""
}

// PackName returns the name part after the scope.
func (c *QlpackConfig) PackName() string {
	if idx := strings.Index(c.Name, "/"); idx >= 0 {
		return c.Name[idx+1:]
	}
	return c.Name
}

// ModuleName converts "foo/cpp-customizations" → "foo.cpp_customizations"
// for use in QL import statements.
func (c *QlpackConfig) ModuleName() string {
	return strings.ReplaceAll(strings.ReplaceAll(c.Name, "-", "_"), "/", ".")
}

// Pack is a resolved CodeQL pack.
type Pack struct {
	YmlPath string
	Config  QlpackConfig
	Kind    PackKind
	Deps    []*Pack
}

// Dir returns the directory containing qlpack.yml.
func (p *Pack) Dir() string { return filepath.Dir(p.YmlPath) }

// CustomizationsPath returns the expected path for Customizations.qll.
func (p *Pack) CustomizationsPath() string {
	return filepath.Join(p.Dir(), "Customizations.qll")
}

// IsCustomizable returns true if Customizations.qll already exists in the pack dir.
func (p *Pack) IsCustomizable() bool {
	_, err := os.Stat(p.CustomizationsPath())
	return err == nil
}

// LockFilePath returns the path to codeql-pack.lock.yml.
func (p *Pack) LockFilePath() string {
	return filepath.Join(p.Dir(), "codeql-pack.lock.yml")
}

// DepsPath returns the path to the .codeql/ installed-dependencies directory.
func (p *Pack) DepsPath() string { return filepath.Join(p.Dir(), ".codeql") }

// CachePath returns the path to the .cache/ directory.
func (p *Pack) CachePath() string { return filepath.Join(p.Dir(), ".cache") }

// loadConfig reads and parses a qlpack.yml.
func loadConfig(ymlPath string) (QlpackConfig, error) {
	data, err := os.ReadFile(ymlPath)
	if err != nil {
		return QlpackConfig{}, fmt.Errorf("reading %s: %w", ymlPath, err)
	}
	var cfg QlpackConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return QlpackConfig{}, fmt.Errorf("parsing %s: %w", ymlPath, err)
	}
	return cfg, nil
}

// classify determines the kind of a pack.
// A pack is a CustomizationPack if it is a library pack AND has a
// <name_underscored>/Customizations.qll file relative to its directory.
// e.g. "foo/cpp-customizations" → check for "foo/cpp_customizations/Customizations.qll"
func classify(ymlPath string, cfg QlpackConfig) PackKind {
	if !cfg.Library {
		return QueryPack
	}
	normalized := strings.ReplaceAll(cfg.Name, "-", "_")
	customPath := filepath.Join(filepath.Dir(ymlPath), normalized, "Customizations.qll")
	if _, err := os.Stat(customPath); err == nil {
		return CustomizationPack
	}
	return LibraryPack
}

// ListPacks runs `codeql pack ls --format=json <dir>` and returns all packs found.
func ListPacks(codeqlBin, dir string) ([]*Pack, error) {
	res, err := executil.NewRunner(codeqlBin).Run("pack", "ls", "--format=json", dir)
	if err != nil {
		return nil, fmt.Errorf("codeql pack ls %s: %w", dir, err)
	}
	raw := res.Stdout
	var parsed struct {
		Packs map[string]interface{} `json:"packs"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parsing pack ls output: %w", err)
	}
	var packs []*Pack
	for ymlPath := range parsed.Packs {
		cfg, err := loadConfig(ymlPath)
		if err != nil {
			return nil, err
		}
		packs = append(packs, &Pack{
			YmlPath: ymlPath,
			Config:  cfg,
			Kind:    classify(ymlPath, cfg),
		})
	}
	return packs, nil
}

// saveConfig writes the in-memory qlpack.yml back to disk, preserving existing
// fields not tracked by QlpackConfig.
func saveConfig(ymlPath string, updates map[string]interface{}) error {
	data, err := os.ReadFile(ymlPath)
	if err != nil {
		return err
	}
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}
	for k, v := range updates {
		raw[k] = v
	}
	out, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}
	return os.WriteFile(ymlPath, out, 0644)
}
