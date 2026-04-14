package pack

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/executil"
	"github.com/trganda/codeql-development-toolkit/internal/utils"
	"gopkg.in/yaml.v3"
)

// PackKind classifies a CodeQL pack.
type PackKind int

const (
	QueryPack         PackKind = iota
	LibraryPack       PackKind = iota
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

// HasExtractor returns true if the pack has an extractor field set.
func (c *QlpackConfig) HasExtractor() bool {
	return c.Extractor != ""
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

func (c *QlpackConfig) FullName() string {
	return c.Name
}

// Pack is a resolved CodeQL pack.
type Pack struct {
	YmlPath string
	Config  QlpackConfig
	Deps    []*Pack
}

// Dir returns the directory containing qlpack.yml.
func (p *Pack) Dir() string { return filepath.Dir(p.YmlPath) }

// IsTestPack returns true if the pack is a test pack, identified by either
// having an extractor field set or being located under a test/ directory.
func (p *Pack) IsTestPack() bool {
	for dir := p.Dir(); dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		if filepath.Base(dir) == "test" {
			return true
		}
	}

	if p.Config.HasExtractor() {
		return true
	}

	return false
}

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

func (p *Pack) CopyTo(destRoot string) (*Pack, error) {
	scope := p.Config.Scope()
	name := p.Config.PackName()
	version := p.Config.Version
	if version == "" {
		version = "0.0.0"
	}
	destDir := filepath.Join(destRoot, scope, name, version)
	slog.Debug("Copying pack", "from", p.Dir(), "to", destDir)
	if err := utils.CopyDir(p.Dir(), destDir); err != nil {
		return nil, fmt.Errorf("copying pack %s: %w", p.Config.Name, err)
	}
	copy := &Pack{
		YmlPath: filepath.Join(destDir, filepath.Base(p.YmlPath)),
		Config:  p.Config,
		Deps:    p.Deps,
	}
	return copy, nil
}

// SaveConfig writes the in-memory qlpack.yml back to disk, preserving existing
// fields not tracked by QlpackConfig.
func (p *Pack) SaveConfig(updates map[string]interface{}) error {
	data, err := os.ReadFile(p.YmlPath)
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
	return os.WriteFile(p.YmlPath, out, 0644)
}

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
		packs = append(packs,
			&Pack{
				YmlPath: ymlPath,
				Config:  cfg,
			})
	}
	return packs, nil
}
