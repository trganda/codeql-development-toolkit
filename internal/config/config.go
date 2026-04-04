package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CodeQLPackConfiguration represents a single CodeQL pack entry in qlt.conf.json.
type CodeQLPackConfiguration struct {
	Name             string `json:"Name"`
	Bundle           bool   `json:"Bundle"`
	Publish          bool   `json:"Publish"`
	ReferencesBundle bool   `json:"ReferencesBundle"`
}

// QLTConfig holds the QLT configuration loaded from qlt.conf.json.
type QLTConfig struct {
	CodeQLCLI              string                    `json:"CodeQLCLI"`
	CodeQLCLIBundle        string                    `json:"CodeQLCLIBundle"`
	CodeQLConfiguration    string                    `json:"CodeQLConfiguration,omitempty"`
	CodeQLPackConfiguration []CodeQLPackConfiguration `json:"CodeQLPackConfiguration,omitempty"`
	base                   string
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
func MustLoadFromFile(base string) (*QLTConfig, error) {
	path := ConfigFilePath(base)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("cannot read values from missing file %s", path)
	}
	return LoadFromFile(base)
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
	return os.WriteFile(ConfigFilePath(base), data, 0644)
}
