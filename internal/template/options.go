package template

// TestInitOptions holds template variables for the test init workflow.
type TestInitOptions struct {
	Language   string // display name used in the workflow title and filename
	LangFlag   string // value for --language flag; empty means test all languages
	Branch     string
	NumThreads int
	UseRunner  string
	CodeqlArgs string
}

// BundleInitOptions holds template variables for bundle init.
type BundleInitOptions struct {
	Languages []string
	Branch    string
	// Packs is the list of pack names from qlt.conf.json with Bundle: true.
	// They are passed to github/codeql-action/init via its `config.packs` input.
	Packs []string
}
