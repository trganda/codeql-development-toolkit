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
                     --pack (repeatable), --num-threads, --codeql-args are persistent flags
                     on the parent and are bundled into utils.CommonFlags before being passed
                     to internal/.
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
- `internal/query` — CodeQL query execution (`RunQuery`), compilation (`RunCompile`), pack dependency installation (`RunPackInstall`), and workspace initialisation (`InitWorkspace`). `RunPackInstall(base, c)` and `RunCompile(base, c)` take `*utils.CommonFlags` and use `pack.SelectPacks` to honour `--pack`. Used by both `cmd/query` and `cmd/phase`.
- `internal/test` — `RunUnitTests(base, c, output)` takes `*utils.CommonFlags` and a report output path. It lists all packs under `base`, filters by `c.Packs` via `pack.SelectPacks`, skips non-test packs (`!IsTestPack()`), then resolves `.qlref` tests per remaining pack via `codeql resolve tests <pack-dir>`. Used by both `cmd/test run` and `cmd/phase test`.
- `internal/pack` — `ListPacks(cli, dir)` runs `codeql pack ls` and returns `[]*Pack`. `Pack.IsTestPack()` returns true when the pack lives under a `test/` directory or declares an extractor. `SelectPacks(allPacks, names, skipTest)` resolves a list of pack names against `allPacks` (matching by full name, then by unique short name) and is the shared filter primitive behind `--pack`. Used by `cmd/pack list`, `cmd/pack resolve`, `cmd/pack publish`, `cmd/phase install`, `cmd/phase compile`, `cmd/phase test`, and `cmd/phase publish`.
- `internal/matrix` — `Build(osVersions, cliVersion)` constructs and marshals a GitHub Actions CI matrix JSON.
- `internal/utils` — `CommonFlags{Packs []string, NumThreads int, CodeQLArgs string}` DTO assembled in `cmd/phase/phase.go` from persistent flags and passed by pointer into every phase subcommand and the `internal/` functions they delegate to. Also hosts `CheckWorkspace`.

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

**Common flags:** `--pack` (repeatable; full or unique short name), `--num-threads` (default 0 = all cores), and `--codeql-args` are persistent flags on the parent `phase` command and inherited by every subcommand. They are bundled into `*utils.CommonFlags` and passed through `cmd/phase/chain.go` into `internal/`. Phase-specific flags (e.g. `--scope`, `--bundle`, `--platform`, `--output`) stay on the individual subcommands.

**Two supported flows (parallel alternatives):**
1. `init → ... → verify → publish`
2. `init → ... → verify → package → publish` (when using custom bundles; `package` is not a prerequisite of `publish`)

The `package` phase is config-driven: it reads `CodeQLPackConfiguration` entries with `Bundle=true` from `qlt.conf.json` — no `--pack` flags required.

The granular commands (`qlt query`, `qlt test`, `qlt pack`, etc.) are preserved for CI use or fine-grained control.

## CI / Release Workflows

The release pipeline lives in `.github/workflows/release.yml`. It triggers on `release: published`, extracts the version from the tag, then fans out via a matrix to the SLSA Level 3 Go builder (`slsa-framework/slsa-github-generator/.github/workflows/builder_go_slsa3.yml@v2.0.0`). One matrix entry per platform; each entry passes a different `.slsa-goreleaser/<platform>.yml` config file:

- `linux-amd64`, `linux-arm64`
- `darwin-amd64`, `darwin-arm64`
- `windows-amd64`, `windows-arm64`

Each config sets `GOOS`/`GOARCH`/`CGO_ENABLED=0`, the `-trimpath` flag, the `cmd.Version` ldflag (sourced from the release tag via `evaluated-envs: VERSION:...`), and a platform-suffixed `binary:` name. Output artifacts are **raw binaries** (e.g. `qlt-linux-amd64`, `qlt-windows-amd64.exe`) plus a `<binary>.intoto.jsonl` SLSA provenance file per platform — no `.zip` wrapping. The builder uploads everything to the GitHub release automatically.

## Git Conversions

Must follow the [Conventional Commits](https://www.conventionalcommits.org/) specification to generate git commit message.