# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make build          # compile to dist/qlt (version from git describe)
make install        # go install to $GOPATH/bin
make test           # go test ./...
make lint           # go vet ./...
make clean          # remove dist/

# Override version at build time
make build VERSION=1.2.3

# Build without make
go build -ldflags "-X github.com/trganda/codeql-development-toolkit/cmd.Version=dev" -o dist/qlt .

# Run a single test package
go test ./internal/config/...
go test ./cmd/query/...
```

## Architecture

**Entry point:** `main.go` → `cmd.Execute()` in `cmd/root.go`.

**Command tree (cobra):** `cmd/root.go` registers global flags and wires all subcommands. Each subcommand lives in its own package under `cmd/`:

```
cmd/
  root.go          — global flags: --base, --automation-type, --development, --verbose
  version.go       — Version var injected via -ldflags at build time
  phase/           — phase init / install / compile / test / verify / package / publish
                     high-level lifecycle phases; each phase runs the full chain from install
                     up to and including the requested phase (Maven-style).
                     --language, --num-threads, --codeql-args are persistent flags on the parent.
  query/           — query generate new-query / run
                     --use-bundle is a persistent flag scoped to this subcommand only
  codeql/          — codeql set version / get version  (auto-resolves from GitHub API)
                     codeql install downloads CLI or bundle based on EnableCustomCodeQLBundles
  test/            — test init / get-matrix / validate
  validation/      — validation run check-queries
  action/          — action init test / action init bundle-test
                     generates GitHub Actions workflows for unit tests and bundle integration tests.
                     test: --language <lang|all>, --branch, --num-threads, --use-runner, --overwrite
                     bundle-test: --language <lang> (repeatable for multi-language), --branch, --overwrite
  bundle/          — bundle init (generates GitHub Actions workflows)
  pack/            — pack list [--all] [--language]
                     pack resolve [--language] — auto-discovers non-test packs and registers them in qlt.conf.json
```

**Shared `--base` flag** points to the target CodeQL repository being managed (not this repo itself). All file writes go relative to `--base`.

## cmd/ vs internal/ boundary

**`cmd/` contains only:**
- Cobra `Command` definitions (`Use`, `Short`, `Long`, `RunE`)
- Flag declarations (`cmd.Flags()`, `MarkFlagRequired`)
- User-facing stdout output (`fmt.Print*`)
- Thin glue that reads flags and calls into `internal/`

**`internal/` contains everything else:**
- Business logic, data transformation, file I/O
- External process invocation (`executil.Runner`)
- Structs used across more than one command
- Any function that could be unit-tested without a cobra context

The rule of thumb: if a function doesn't reference `*cobra.Command` or flag variables, it belongs in `internal/`. A `cmd/` file that grows beyond ~50 lines of non-flag code is a signal that logic should be extracted.

**`internal/` packages:**

- `internal/config` — reads/writes `qlt.conf.json` (`QLTConfig` struct). Fields: `CodeQLCLIVersion` (json:`"version"`), `CodeQLPackConfiguration` (json:`"packs"`) — a slice of `{Name, Bundle, Publish, ReferencesBundle}`. `LoadFromFile` returns nil if missing; `MustLoadFromFile` exits on error. `UpsertPackConfig(name, bundle)` adds or updates an entry (skips duplicates by name).
- `internal/template` — `Render` and `WriteFile` backed by `text/template` with `[[ ]]` delimiters (avoids conflict with GitHub Actions `${{ }}` syntax). All template files are embedded via `//go:embed` in `internal/template/embed.go`. Available template functions: `toLower`, `join` (wraps `strings.Join`).
- `internal/release` — resolves latest CodeQL versions from the GitHub API (`github/codeql-cli-binaries` and `github/codeql-action`). 5s timeout, falls back to hardcoded constants (`FallbackCLIVersion`, `FallbackBundleVersion`).
- `internal/log` — wraps `log/slog`. `Init(verbose bool)` is called from `PersistentPreRunE`. Without `--verbose`: compact format (no timestamps), Info level. With `--verbose`: full text handler, Debug level. Convention: `slog.Debug` for traces, `slog.Info` for lifecycle events, `fmt.Print*` for user-facing stdout output only.
- `internal/executil` — thin wrapper around `os/exec`. `NewRunner(binary)` returns a `Runner` that captures stdout/stderr into a `Result`. On non-zero exit, `Run` returns a `*RunError` (implements `error` and `Unwrap`) carrying the binary, args, exit code, and trimmed stderr. Callers check `res.Stdout`/`res.Stderr` directly or use the `StdoutString()`/`StderrString()` convenience methods.
- `internal/language` — helpers mapping language names to directories (`c`/`cpp` → `"cpp"`), CodeQL import names, and source file extensions.
- `internal/paths` — content-addressed path layout under `$HOME/.qlt/`. All versioned directories use an MD5 hash of the version string. Key functions: `CLIInstallDir`, `BundleInstallDir`, `CustomBundlePath`, `BundleArchivePath`, `ResolveCodeQLBinary`.
- `internal/codeql` — CLI/bundle download, checksum verification, platform detection, and extraction. `Install(base, version, platform)` is the single entry point used by `cmd/codeql install`.
- `internal/query` — CodeQL query execution (`RunQuery`), compilation (`RunCompile`), pack dependency installation (`RunPackInstall`), and workspace initialisation (`InitWorkspace`). Used by both `cmd/query` and `cmd/phase`.
- `internal/test` — `RunUnitTests(base, lang, codeqlArgs, reportOutput, numThreads)` resolves and runs all `.qlref` test files. When `lang` is `""` or `"all"`, tests are resolved from `base`; otherwise from `base/<lang-dir>`. Used by both `cmd/test run` and `cmd/phase test`.
- `internal/pack` — `ListPacks(cli, dir)` runs `codeql pack ls` and returns `[]*Pack`. `Pack.IsTestPack()` returns true when the pack lives under a `test/` directory or declares an extractor. Used by `cmd/pack list`, `cmd/pack resolve`, `cmd/pack publish`, and `cmd/phase publish`.
- `internal/matrix` — `Build(osVersions, cliVersion)` constructs and marshals a GitHub Actions CI matrix JSON.

## Logger

The `internal/log` package wraps `log/slog`. Alwasys use the logger to provide structured logs with context (e.g. `slog.Info("Installed CodeQL CLI", "version", version, "path", path)`). Use `fmt.Print*` only for user-facing output that should not be treated as logs.

## Templates

Templates live under `internal/template/files/` and are embedded at compile time. Template subdirectories map to features:

- `query/<lang>/` — `new-query.tmpl`, `new-dataflow-query.tmpl`, `qlpack-query.tmpl`, `qlpack-test.tmpl`, `test.tmpl`, `expected.tmpl`
- `query/all/testref.tmpl`, `query/codeql-workspace.tmpl`
- `test/actions/`, `bundle/actions/`, `validation/actions/`

**Delimiter:** `[[ ]]` not `{{ }}`. Use `[[- ]]` / `[[ -]]` for whitespace trimming. Available functions: `toLower`, `join` (e.g. `[[ join .Languages ", " ]]`).

## Path Layout

All versioned directories under `$HOME/.qlt/` use an MD5 hash of the version/bundle string:

```
$HOME/.qlt/
├── packages/<md5(cliVersion)>/         ← extracted CLI  (codeql install, default)
│   ├── codeql/
│   ├── codeql-<platform>.zip
│   └── codeql-<platform>.zip.checksum.txt
├── bundle/<md5(bundleName)>/           ← extracted bundle  (codeql install, EnableCustomCodeQLBundles=true)
│   ├── codeql/
│   ├── codeql-bundle[-platform].tar.gz
│   └── codeql-bundle[-platform].tar.gz.checksum.txt
└── custom-bundle/<md5(bundleName)>/    ← output of `qlt bundle create`
    └── codeql-bundle.tar.gz
```

`ResolveCodeQLBinary` checks `EnableCustomCodeQLBundles` in config: if true it uses the bundle binary, otherwise the CLI binary, falling back to `codeql` on `PATH`.

## Configuration File

`qlt.conf.json` is written to `--base` by `qlt codeql set version`. Key fields:

```json
{
  "version": "2.25.1",
  "packs": [
    { "name": "scope/pack-name", "bundle": true, "publish": false, "referencesBundle": false }
  ]
}
```

- `version` (`CodeQLCLIVersion`) — CodeQL CLI version string; used by `ResolveCodeQLBinary` and install commands.
- `packs` (`CodeQLPackConfiguration`) — upserted by `qlt query generate new-query` and `qlt pack resolve`; `bundle: true` only when `--use-bundle` is set.

## Lifecycle

`qlt phase` provides a Maven-inspired, phase-oriented workflow on top of the granular commands. Each phase delegates entirely to an `internal/` package — no business logic lives in `cmd/phase/`.

| Phase | Command | Delegates to | Status |
|---|---|---|---|
| initialize | `qlt phase init` | `internal/query.InitWorkspace` | implemented |
| install | `qlt phase install` | `internal/query.RunPackInstall` | implemented |
| compile | `qlt phase compile` | `internal/query.RunCompile` | implemented |
| test | `qlt phase test` | `internal/test.RunUnitTests` | implemented |
| verify | `qlt phase verify` | — | placeholder |
| package | `qlt phase package` | `internal/bundle.Create` | implemented |
| publish | `qlt phase publish` | `internal/pack.FindQlpacks` + codeql publish | implemented |

**Phase chaining:** Every phase except `init` runs the full chain from `install` up to and including the requested phase. For example, `qlt phase test` runs install → compile → test automatically. `init` is never auto-run — it must be invoked explicitly.

**Workspace guard:** The parent `phase` command has a `PersistentPreRunE` that runs `utils.CheckWorkspace` for every subcommand except `init`. If `codeql-workspace.yml` is missing, non-init phases fail with guidance to run `phase init` first.

**Common flags:** `--language`, `--num-threads` (default 0 = all cores), and `--codeql-args` are persistent flags on the parent `phase` command and inherited by every subcommand. Phase-specific flags (e.g. `--scope`, `--bundle`, `--platform`) stay on the individual subcommands.

**Two supported flows (parallel alternatives):**
1. `init → ... → verify → publish`
2. `init → ... → verify → package → publish` (when using custom bundles; `package` is not a prerequisite of `publish`)

The `package` phase is config-driven: it reads `CodeQLPackConfiguration` entries with `Bundle=true` from `qlt.conf.json` — no `--pack` flags required.

The granular commands (`qlt query`, `qlt test`, `qlt pack`, etc.) are preserved for CI use or fine-grained control.

## CI / Release Workflows

The release pipeline is split into three platform workflows called from `internal-release-build.yml`:

- `internal-build-release-linux64.yml` — `ubuntu-latest`, produces `qlt-linux-x86_64.zip`
- `internal-build-release-macos64.yml` — `macos-26` (arm64), produces `qlt-macos-arm64.zip`
- `internal-build-release-win64.yml` — `windows-latest`, produces `qlt-windows-x64.zip`

All three use `actions/setup-go@v6` with the version from `go.mod`, inject `inputs.version` via `-ldflags`, and upload to the GitHub release via `gh release upload`.

## Git Conversions

Must follow the [Conventional Commits](https://www.conventionalcommits.org/) specification to generate git commit message.