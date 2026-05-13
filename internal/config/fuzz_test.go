package config

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzLoadFromFile feeds arbitrary bytes to LoadFromFile. The contract is that
// parsing must never panic and must either return a *QLTConfig or an error.
func FuzzLoadFromFile(f *testing.F) {
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"version":"2.25.1","packs":[{"name":"foo/bar","bundle":true}]}`))
	f.Add([]byte(`{"version":1}`))
	f.Add([]byte(`not json`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		base := t.TempDir()
		if err := os.WriteFile(filepath.Join(base, "qlt.conf.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
		cfg, err := LoadFromFile(base)
		if err != nil {
			return
		}
		if cfg == nil {
			return
		}
		// Round-trip: marshal and re-unmarshal must succeed.
		out := t.TempDir()
		if err := cfg.SaveToFile(out); err != nil {
			t.Fatalf("save: %v", err)
		}
		if _, err := LoadFromFile(out); err != nil {
			t.Fatalf("reload after save: %v", err)
		}
	})
}
