package template

import "embed"

//go:embed files
var fs embed.FS

// Get returns the content of an embedded template file.
// path is relative to the files/ directory, e.g. "query/cpp/new-query.tmpl".
func Get(path string) (string, error) {
	data, err := fs.ReadFile("files/" + path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
