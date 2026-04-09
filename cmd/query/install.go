package query

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/language"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// newInstallCmd returns `query install`.
func newInstallCmd(base *string) *cobra.Command {
	var lang, pack string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install dependencies for a query pack",
		Long: `Install dependencies declared in a query pack's qlpack.yml using 'codeql pack install'.

The qlpack.yml is located at <base>/<language>/<pack>/src/qlpack.yml.
If --language and --pack are omitted, every qlpack.yml found under <base> is installed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing query install command", "base", *base, "language", lang, "pack", pack)
			return runPackInstall(*base, lang, pack)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Language of the query pack (e.g. go, java)")
	cmd.Flags().StringVar(&pack, "pack", "", "Pack name to install dependencies for")
	return cmd
}

func runPackInstall(base, lang, pack string) error {
	codeql, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return err
	}

	var qlpacks []string

	if lang != "" && pack != "" {
		langDir := language.ToDirectory(lang)
		p := fmt.Sprintf("%s/%s/%s/src/qlpack.yml", base, langDir, pack)
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("qlpack.yml not found at %s", p)
		}
		qlpacks = append(qlpacks, p)
	} else {
		qlpacks, err = findQlpackFiles(base)
		if err != nil {
			return fmt.Errorf("search for qlpack.yml files: %w", err)
		}
		if len(qlpacks) == 0 {
			return fmt.Errorf("no qlpack.yml files found under %s", base)
		}
	}

	for _, qlpack := range qlpacks {
		slog.Info("Installing pack dependencies", "qlpack", qlpack)
		fmt.Printf("Running: %s pack install %s\n", codeql, qlpack)
		cmd := exec.Command(codeql, "pack", "install", qlpack)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return fmt.Errorf("codeql pack install exited with code %d for %s", exitErr.ExitCode(), qlpack)
			}
			return fmt.Errorf("run codeql pack install: %w", err)
		}
	}
	return nil
}

// findQlpackFiles walks base up to 4 levels deep and collects all qlpack.yml paths.
func findQlpackFiles(base string) ([]string, error) {
	var found []string
	var walk func(string, int)
	walk = func(dir string, depth int) {
		if depth > 4 {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			path := dir + "/" + e.Name()
			if !e.IsDir() && e.Name() == "qlpack.yml" {
				found = append(found, path)
				continue
			}
			if e.IsDir() {
				walk(path, depth+1)
			}
		}
	}
	walk(base, 0)
	return found, nil
}
