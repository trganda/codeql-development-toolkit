package template

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var funcMap = template.FuncMap{
	"toLower": strings.ToLower,
	"join":    strings.Join,
}

// Render executes a template string with [[ ]] delimiters and the given data.
// [[ ]] delimiters are used to avoid conflicts with GitHub Actions {{ }} syntax.
func Render(tmplContent string, data any) (string, error) {
	t, err := template.New("").Delims("[[", "]]").Funcs(funcMap).Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

// WriteFile renders a template string and writes the output to dst.
// If overwrite is false and dst already exists, the write is skipped.
func WriteFile(tmplContent, dst string, data any, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(dst); err == nil {
			return nil // already exists, skip
		}
	}

	content, err := Render(tmplContent, data)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create directories for %s: %w", dst, err)
	}

	return os.WriteFile(dst, []byte(content), 0644)
}
