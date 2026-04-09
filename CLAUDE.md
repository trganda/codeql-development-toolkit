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
  root.go          — global flags: --base, --automation-type, --development, --use-bundle, --verbose
  version.go       — Version var injected via -ldflags at build time
  query/           — query init / generate new-query / run install-packs
  codeql/          — codeql set version / get version  (auto-resolves from GitHub API)
  test/            — test init / run get-matrix / run execute-unit-tests / run validate-unit-tests
  validation/      — validation run check-queries
  bundle/          — bundle init / set enable|disable / get enabled-bundles / run validate-integration-tests
  pack/            — pack run hello (placeholder)
```

**Shared `--base` flag** points to the target CodeQL repository being managed (not this repo itself). All file writes go relative to `--base`.

**`internal/` packages:**

- `internal/config` — reads/writes `qlt.conf.json` (`QLTConfig` struct with `CodeQLCLI` and `CodeQLCLIBundle` fields). `LoadFromFile` returns nil if missing; `MustLoadFromFile` errors.
- `internal/template` — `Render` and `WriteFile` backed by `text/template` with `[[ ]]` delimiters (avoids conflict with GitHub Actions `${{ }}` syntax). All template files are embedded via `//go:embed` in `internal/template/embed.go`.
- `internal/release` — resolves latest CodeQL versions from the GitHub API (`github/codeql-cli-binaries` and `github/codeql-action`). 5s timeout, falls back to hardcoded constants (`FallbackCLIVersion`, `FallbackBundleVersion`).
- `internal/log` — wraps `log/slog`. `Init(verbose bool)` is called from `PersistentPreRunE`. Without `--verbose`: compact format (no timestamps), Info level. With `--verbose`: full text handler, Debug level. Convention: `slog.Debug` for traces, `slog.Info` for lifecycle events, `fmt.Print*` for user-facing stdout output only.
- `internal/language` — helpers mapping language names to directories (`c`/`cpp` → `"cpp"`), CodeQL import names, and source file extensions.

## Templates

Templates live under `internal/template/files/` and are embedded at compile time. Template subdirectories map to features:

- `query/<lang>/` — `new-query.tmpl`, `new-dataflow-query.tmpl`, `qlpack-query.tmpl`, `qlpack-test.tmpl`, `test.tmpl`, `expected.tmpl`
- `query/all/testref.tmpl`, `query/codeql-workspace.tmpl`
- `test/actions/`, `bundle/actions/`, `validation/actions/`

**Delimiter:** `[[ ]]` not `{{ }}`. Use `[[- ]]` / `[[ -]]` for whitespace trimming. The `toLower` function is available in all templates.

## Configuration File

`qlt.conf.json` is written to `--base` by `qlt codeql set version`. Key fields:

```json
{
  "CodeQLCLI": "2.25.1",
  "CodeQLCLIBundle": "codeql-bundle-v2.25.1"
}
```

## CI / Release Workflows

The release pipeline is split into three platform workflows called from `internal-release-build.yml`:

- `internal-build-release-linux64.yml` — `ubuntu-latest`, produces `qlt-linux-x86_64.zip`
- `internal-build-release-macos64.yml` — `macos-14` (arm64), produces `qlt-macos-arm64.zip`
- `internal-build-release-win64.yml` — `windows-latest`, produces `qlt-windows-x64.zip`

All three use `actions/setup-go@v5` with the version from `go.mod`, inject `inputs.version` via `-ldflags`, and upload to the GitHub release via `gh release upload`.

## Git Conversions

Must follow the [Conventional Commits](https://www.conventionalcommits.org/) specification to generate git commit message.