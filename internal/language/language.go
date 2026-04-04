package language

import "strings"

// SupportedLanguages lists all CodeQL-supported languages.
var SupportedLanguages = []string{
	"c", "cpp", "csharp", "go", "java", "javascript", "python", "ruby",
}

// ToDirectory maps a language option string to its CodeQL directory name.
func ToDirectory(lang string) string {
	switch strings.ToLower(lang) {
	case "c", "cpp":
		return "cpp"
	default:
		return strings.ToLower(lang)
	}
}

// ToImport maps a language to its CodeQL import name.
func ToImport(lang string) string {
	switch strings.ToLower(lang) {
	case "c", "cpp":
		return "cpp"
	case "csharp":
		return "csharp"
	case "go":
		return "go"
	case "java":
		return "java"
	case "javascript":
		return "javascript"
	case "python":
		return "python"
	case "ruby":
		return "ruby"
	default:
		return lang
	}
}

// ToExtension maps a language to its source file extension.
func ToExtension(lang string) string {
	switch strings.ToLower(lang) {
	case "c":
		return "c"
	case "cpp":
		return "cpp"
	case "csharp":
		return "cs"
	case "go":
		return "go"
	case "java":
		return "java"
	case "javascript":
		return "js"
	case "python":
		return "py"
	case "ruby":
		return "rb"
	default:
		return lang
	}
}

// ToSafeTestName replaces hyphens with underscores in a test name.
func ToSafeTestName(lang, name string) string {
	return strings.ReplaceAll(name, "-", "_")
}
