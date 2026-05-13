package bundle

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzReadSarifResults feeds arbitrary bytes to the SARIF result reader. The
// contract is that parsing must never panic and must either return results or
// an error.
func FuzzReadSarifResults(f *testing.F) {
	f.Add([]byte(`{"runs":[]}`))
	f.Add([]byte(`{"runs":[{"results":[{"ruleId":"x","message":{"text":"m"}}]}]}`))
	f.Add([]byte(`{"runs":null}`))
	f.Add([]byte(`not sarif`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		path := filepath.Join(dir, "in.sarif")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		_, _ = readSarifResults(path)
	})
}
