package query

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/language"
)

// newCompileCmd returns `query compile`.
func newCompileCmd(base *string) *cobra.Command {
	var lang, pack string
	var threads int
	cmd := &cobra.Command{
		Use:   "compile",
		Short: "Compile CodeQL query files (.ql and .qll)",
		Long: `Compile all .ql and .qll files using 'codeql query compile'.

Files are searched under <base>/<language>/<pack>/src/.
If --language and --pack are omitted, all query files found under <base> are compiled.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing query compile command", "base", *base, "language", lang, "pack", pack)
			return runCompile(*base, lang, pack, threads)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Language of the query pack (e.g. go, java)")
	cmd.Flags().StringVar(&pack, "pack", "", "Pack name to compile")
	cmd.Flags().IntVar(&threads, "threads", 0, "Number of threads (0 = use all available cores)")
	return cmd
}

func runCompile(base, lang, pack string, threads int) error {
	codeql, err := resolveCodeQLBinary()
	if err != nil {
		return err
	}

	var searchRoot string
	if lang != "" && pack != "" {
		langDir := language.ToDirectory(lang)
		searchRoot = filepath.Join(base, langDir, pack, "src")
		if _, err := os.Stat(searchRoot); err != nil {
			return fmt.Errorf("directory not found: %s", searchRoot)
		}
	} else {
		searchRoot = base
	}

	files, err := findQueryFiles(searchRoot)
	if err != nil {
		return fmt.Errorf("search for query files: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no .ql files found under %s", searchRoot)
	}

	slog.Info("Compiling query files", "count", len(files), "root", searchRoot)

	args := []string{"query", "compile"}
	if threads != 0 {
		args = append(args, fmt.Sprintf("--threads=%d", threads))
	}
	args = append(args, "--")
	args = append(args, files...)

	fmt.Printf("Running: %s %s\n", codeql, strings.Join(args, " "))
	cmd := exec.Command(codeql, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("codeql query compile exited with code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("run codeql query compile: %w", err)
	}
	return nil
}

// findQueryFiles walks dir recursively and collects all .ql and .qll file paths.
func findQueryFiles(dir string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			ext := filepath.Ext(d.Name())
			if ext == ".ql" {
				found = append(found, path)
			}
		}
		return nil
	})
	return found, err
}
