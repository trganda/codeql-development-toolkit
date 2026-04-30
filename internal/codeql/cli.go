package codeql

import (
	"fmt"
)

// CLI is a typed wrapper around the codeql binary. It owns an Runner
// and exposes one method per codeql subcommand that qlt invokes.
type CLI struct {
	binary string
	runner *Runner
}

func NewCLI(binary string) *CLI {
	return &CLI{binary: binary, runner: NewRunner(binary)}
}

func (c *CLI) Binary() string { return c.binary }

// Run is an escape hatch for codeql invocations not yet modelled here.
func (c *CLI) Run(args ...string) (*Result, error) {
	return c.runner.Run(args...)
}

// PackLs runs `codeql pack ls --format=json <dir>`.
func (c *CLI) PackLs(dir string) (*Result, error) {
	return c.runner.Run("pack", "ls", "--format=json", dir)
}

// PackInstall runs `codeql pack install --format=json [--common-caches=<caches>] <target>`.
func (c *CLI) PackInstall(target, commonCaches string) (*Result, error) {
	args := []string{"pack", "install", "--format=json"}
	if commonCaches != "" {
		args = append(args, "--common-caches="+commonCaches)
	}
	args = append(args, target)
	return c.runner.Run(args...)
}

// PackResolveDependencies runs `codeql pack resolve-dependencies --format=json <dir>`.
func (c *CLI) PackResolveDependencies(dir string) (*Result, error) {
	return c.runner.Run("pack", "resolve-dependencies", "--format=json", dir)
}

// ResolvePacks runs `codeql resolve packs --format=json`.
func (c *CLI) ResolvePacks() (*Result, error) {
	return c.runner.Run("resolve", "packs", "--format=json")
}

// PackCreate runs `codeql pack create --format=json --output=<output> [--common-caches=<caches>] <dir>`.
func (c *CLI) PackCreate(dir, output, commonCaches string) (*Result, error) {
	args := []string{"pack", "create", "--format=json", "--output=" + output}
	if commonCaches != "" {
		args = append(args, "--common-caches="+commonCaches)
	}
	args = append(args, dir)
	return c.runner.Run(args...)
}

// PackBundle runs `codeql pack bundle --format=json --pack-path=<output> [--common-caches=<caches>] <dir>`.
func (c *CLI) PackBundle(dir, output, commonCaches string) (*Result, error) {
	args := []string{"pack", "bundle", "--format=json", "--pack-path=" + output}
	if commonCaches != "" {
		args = append(args, "--common-caches="+commonCaches)
	}
	args = append(args, dir)
	return c.runner.Run(args...)
}

// PackPublish runs `codeql pack publish <dir>`.
func (c *CLI) PackPublish(dir string) (*Result, error) {
	return c.runner.Run("pack", "publish", "--format=json", dir)
}

// QueryCompile runs `codeql query compile [--threads=N] -- <files>`.
func (c *CLI) QueryCompile(threads int, files ...string) (*Result, error) {
	args := []string{"query", "compile", "--format=json"}
	if threads != 0 {
		args = append(args, fmt.Sprintf("--threads=%d", threads))
	}

	args = append(args, files...)
	return c.runner.Run(args...)
}

// DatabaseAnalyzeOptions collects the flags used by RunQuery.
type DatabaseAnalyzeOptions struct {
	Database        string
	QueryFile       string
	Format          string
	Output          string
	Threads         int
	AdditionalPacks string
}

// DatabaseAnalyze runs `codeql database analyze --format=... --output=... --threads=N --rerun [--additional-packs=...] <db> <query>`.
func (c *CLI) DatabaseAnalyze(opts DatabaseAnalyzeOptions) (*Result, error) {
	args := []string{
		"database", "analyze",
		"--format=" + opts.Format,
		"--output=" + opts.Output,
		fmt.Sprintf("--threads=%d", opts.Threads),
		"--rerun",
	}
	if opts.AdditionalPacks != "" {
		args = append(args, "--additional-packs="+opts.AdditionalPacks)
	}
	args = append(args, opts.Database, opts.QueryFile)
	return c.runner.Run(args...)
}

// ResolveLanguages runs `codeql resolve languages --format=json`.
func (c *CLI) ResolveLanguages() (*Result, error) {
	return c.runner.Run("resolve", "languages", "--format=json")
}

// ResolveTests runs `codeql resolve tests --strict-test-discovery --format json <dir>`.
func (c *CLI) ResolveTests(dir string) (*Result, error) {
	return c.runner.Run("resolve", "tests", "--strict-test-discovery", "--format=json", dir)
}

// TestRun runs `codeql test run --threads N --format betterjson --quiet [extraArgs] <testFile>`.
func (c *CLI) TestRun(threads int, extraArgs, testFile string) (*Result, error) {
	args := []string{"test", "run", "--threads", fmt.Sprintf("%d", threads), "--format=json", "--quiet"}
	if extraArgs != "" {
		args = append(args, extraArgs)
	}
	args = append(args, testFile)
	return c.runner.Run(args...)
}
