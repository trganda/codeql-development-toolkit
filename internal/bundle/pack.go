package bundle

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/executil"
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
	if err := copyDir(p.Dir(), destDir); err != nil {
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
func ListPacks(codeqlBin, dir string) ([]PackProcessor, error) {
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
	var packs []PackProcessor
	for ymlPath := range parsed.Packs {
		cfg, err := loadConfig(ymlPath)
		if err != nil {
			return nil, err
		}
		packs = append(packs, newPackProcessor(
			&Pack{
				YmlPath: ymlPath,
				Config:  cfg,
			},
			classify(ymlPath, cfg),
		))
	}
	return packs, nil
}

// PackProcessor encapsulates the build-time handling of a single workspace
// pack while assembling a custom CodeQL bundle. Each pack kind has its own
// implementation, selected once at pack-discovery time so the orchestrator
// loop in CustomBundle.Create can call Process without further dispatch.
type PackProcessor interface {
	Process(cb *CustomBundle) error
	GetPack() *Pack
	GetKind() PackKind
}

// newPackProcessor returns the PackProcessor matching p.Kind. It is the only
// place in the package that branches on PackKind; once a Pack has been
// classified, behaviour is dispatched through the interface.
func newPackProcessor(p *Pack, kind PackKind) PackProcessor {
	switch kind {
	case CustomizationPack:
		return &customizationPack{pack: p}
	case LibraryPack:
		return &libraryPack{pack: p}
	default:
		return &queryPack{pack: p}
	}
}

// queryPack runs the simplified 6-step bundling flow for a query
// pack: copy → pack install → pack create. Resolved dependencies are folded
// into the bundle by CustomBundle.Create after every processor has run.
type queryPack struct {
	pack *Pack
	kind PackKind
}

func (q *queryPack) Process(cb *CustomBundle) error {
	p := q.pack
	slog.Info("Processing query pack", "pack", p.Config.Name)

	packCopy, err := p.CopyTo(filepath.Join(cb.tmpDir, "temp"))
	if err != nil {
		return err
	}

	runner := executil.NewRunner(cb.tmpCodeQLBin)

	if _, err := runner.Run(
		"pack", "install",
		"--format=json",
		fmt.Sprintf("--common-caches=%s", cb.commonCachesDir),
		packCopy.Dir(),
	); err != nil {
		return fmt.Errorf("codeql pack install %s: %w", p.Config.Name, err)
	}

	if _, err := runner.Run(
		"pack", "create",
		"--format=json",
		fmt.Sprintf("--output=%s", cb.tmpQlPacksDir),
		fmt.Sprintf("--common-caches=%s", cb.commonCachesDir),
		packCopy.Dir(),
	); err != nil {
		return fmt.Errorf("codeql pack create %s: %w", p.Config.Name, err)
	}

	return nil
}

func (q *queryPack) GetPack() *Pack {
	return q.pack
}

func (q *queryPack) GetKind() PackKind {
	return q.kind
}

// customizationPack is a placeholder for customization-kind packs.
// Injecting Customizations.qll imports into stdlib packs and topologically
// re-bundling is not yet implemented; the pack is skipped with a warning.
type customizationPack struct {
	pack *Pack
	kind PackKind
}

func (c *customizationPack) Process(_ *CustomBundle) error {
	slog.Warn("Customization packs are not yet supported in bundle create; skipping",
		"pack", c.pack.Config.Name)
	return nil
}

func (c *customizationPack) GetPack() *Pack {
	return c.pack
}

func (c *customizationPack) GetKind() PackKind {
	return c.kind
}

// libraryPack is a placeholder for library-kind packs. Library
// bundling is not yet implemented; the pack is skipped with a warning.
type libraryPack struct {
	pack *Pack
	kind PackKind
}

func (l *libraryPack) Process(_ *CustomBundle) error {
	slog.Warn("Library packs are not yet supported in bundle create; skipping",
		"pack", l.pack.Config.Name)
	return nil
}

func (l *libraryPack) GetPack() *Pack {
	return l.pack
}

func (l *libraryPack) GetKind() PackKind {
	return l.kind
}
