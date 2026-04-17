package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

// CodeQLPackConfiguration represents a single CodeQL pack entry in qlt.conf.json.
type CodeQLPackConfiguration struct {
	Name             string `json:"name"`
	Bundle           bool   `json:"bundle"`
	Publish          bool   `json:"publish"`
	ReferencesBundle bool   `json:"referencesBundle"`
}

// QueryEntry records a generated query so it can be resolved by name later.
type QueryEntry struct {
	Name     string `json:"name"`
	Language string `json:"language"`
	Pack     string `json:"pack"`
	Scope    string `json:"scope,omitempty"`
}

// QLTConfig holds the QLT configuration loaded from qlt.conf.json.
type QLTConfig struct {
	CodeQLCLI               string                    `json:"codeQLCLI"`
	CodeQLConfiguration     string                    `json:"codeQLConfiguration,omitempty"`
	Scope                   string                    `json:"scope,omitempty"`
	CodeQLPackConfiguration []CodeQLPackConfiguration `json:"codeQLPackConfiguration,omitempty"`
	base                    string
}

// UpsertPackConfig adds or updates the CodeQLPackConfiguration entry for the given name.
func (c *QLTConfig) UpsertPackConfig(name string, bundle bool) {
	for i, p := range c.CodeQLPackConfiguration {
		if p.Name == name {
			c.CodeQLPackConfiguration[i].Bundle = bundle
			return
		}
	}
	c.CodeQLPackConfiguration = append(c.CodeQLPackConfiguration, CodeQLPackConfiguration{
		Name:   name,
		Bundle: bundle,
	})
}

// ConfigFilePath returns the path to qlt.conf.json under the base directory.
func ConfigFilePath(base string) string {
	return filepath.Join(base, "qlt.conf.json")
}

// LoadFromFile reads qlt.conf.json from the given base directory.
// Returns nil if the file does not exist.
func LoadFromFile(base string) (*QLTConfig, error) {
	path := ConfigFilePath(base)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var c QLTConfig
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	c.base = base
	return &c, nil
}

// MustLoadFromFile loads the config or exits with an error.
func MustLoadFromFile(base string) *QLTConfig {
	path := ConfigFilePath(base)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		slog.Error("qlt.conf.json not found on workspace, run qlt phase init to create it first.", "path", path)
		os.Exit(1)
	}

	c, err := LoadFromFile(base)
	if err != nil {
		slog.Error("Failed to load qlt.conf.json", "path", path, "error", err)
		os.Exit(1)
	}
	return c
}

// SaveToFile writes the config to qlt.conf.json in the base directory,
// creating the directory if it does not exist.
func (c *QLTConfig) SaveToFile(base string) error {
	if err := os.MkdirAll(base, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	dir := ConfigFilePath(base)
	if err := os.WriteFile(dir, data, 0644); err != nil {
		return err
	}

	slog.Info("Saved qlt.conf.json", "path", dir)
	return nil
}
