<div align="center">
<img src="assets/qlt-logo2.png">
</div>

# The CodeQL Development Toolkit (QLT)

QLT makes common CodeQL development workflows easier. Key features:

- Scaffolding and management of CodeQL queries, packs, and unit tests.
- A Maven-inspired lifecycle (`init → install → compile → test → verify → package → publish`) on top of the `codeql` CLI.
- Custom CodeQL bundle creation with per-platform filtering.
- Download and version management of the `codeql` CLI and official bundles.
- GitHub Actions workflow scaffolding for query testing, validation, and bundle integration tests.

> **Go rewrite.** The upstream [.NET toolkit](https://github.com/advanced-security/codeql-development-toolkit) looks like no longer maintained. This repository is a ground-up rewrite in Go with a `cobra`-based CLI. The command surface has changed — see [Commands](#commands) below.

# Installation

**From a release archive.** Grab the archive for your platform from the releases page, unpack it, and put `qlt` on your `PATH`. Builds are published for:

- `qlt-linux-x86_64.zip`
- `qlt-macos-arm64.zip`
- `qlt-windows-x64.zip`

**With `go install`.** Requires Go 1.22+.

```bash
go install github.com/trganda/codeql-development-toolkit/cmd/qlt@latest
```

**From source.**

```
make build             # writes dist/qlt (version from git describe)
make install           # go install into $GOPATH/bin
make build VERSION=1.2.3   # override the embedded version
```

# Usage

```
QLT helps you develop, test, and validate CodeQL queries.

Usage:
  qlt [command]

Available Commands:
  action      Custom CodeQL action management commands
  bundle      Custom CodeQL bundle management commands
  codeql      CodeQL version management commands
  pack        CodeQL pack management commands
  phase       CodeQL development phases
  query       Query feature commands
  test        Unit testing commands
  version     Get the current tool version

Flags:
      --base string   Base repository path (default ".")
      --verbose       Enable verbose logging
```

The `--base` flag points at the **target CodeQL query repository** being managed, not at this toolkit. Every file QLT writes lands relative to `--base`.

# Commands

| Group     | Purpose                                                                           |
|-----------|-----------------------------------------------------------------------------------|
| `phase`   | Maven-style lifecycle: `init`, `install`, `compile`, `test`, `verify`, `package`, `publish` |
| `query`   | Run individual queries (`qlt query run`)                                          |
| `pack`    | Pack scaffolding, listing, validation, and publication (`generate`, `list`, `set`, `run`, `validate`) |
| `codeql`  | Manage the CodeQL CLI / bundle: `install`, `set version`, `get version`           |
| `test`    | Unit-test helpers (`get-matrix` for CI)                                           |
| `bundle`  | Create custom CodeQL bundles and run bundle-related commands                      |
| `action`  | Scaffold GitHub Actions workflows (`init-test`, `init-bundle-test`)               |

# Phase Lifecycle

`qlt phase` mirrors Maven's build lifecycle. Every phase (except `init`) runs the full chain up to and including itself, so `qlt phase test` implicitly runs install → compile → test.

| Phase      | Command              | Notes                                                    |
|------------|----------------------|----------------------------------------------------------|
| initialize | `qlt phase init`     | Writes `codeql-workspace.yml` and `qlt.conf.json`        |
| install    | `qlt phase install`  | `codeql pack install`                                    |
| compile    | `qlt phase compile`  | `codeql query compile`                                   |
| test       | `qlt phase test`     | Resolves `.qlref` tests, runs them, optionally writes a JSON report |
| verify     | `qlt phase verify`   | Placeholder (query metadata / quality checks)            |
| package    | `qlt phase package`  | Builds a custom bundle from packs marked `Bundle=true`   |
| publish    | `qlt phase publish`  | `codeql pack publish` for each pack under `--base`       |

Persistent flags on the parent `phase` command: `--language`, `--num-threads` (0 = all cores), `--codeql-args`.

Two supported flows:

1. `init → install → compile → test → verify → publish`
2. `init → … → verify → package → publish` (when shipping a custom bundle)

> **Note:** install was indenpendent phase. If you need to install the dpendencies of qlpack, you should run `qlt phase install` directly.

# Quickstart

Set up and exercise a CodeQL query repository in four steps.

**1. Install the CodeQL CLI** (downloads into `$HOME/.qlt/packages/<md5(version)>/`):

```
qlt codeql install
```

**2. Initialize the workspace** inside your CodeQL query repo:

```
qlt phase init --scope my-org
```

This writes `codeql-workspace.yml` and `qlt.conf.json` at `--base` (defaults to `.`).

**3. Scaffold a new query pack and query:**

```
qlt pack generate new-query \
  --language java \
  --pack my-queries \
  --query-name MyFirstQuery
```

**4. Run the full chain up through unit tests:**

```
qlt phase test --language java
```

A JSON test report is written to `<base>/target/test/test-report-<timestamp>.json` when `--output` is supplied. Use `--output <file>` to override the path, or omit the flag to skip report generation.

## Custom bundles

`qlt phase package` reads `codeQLPackConfiguration` entries with `bundle=true` and produces a bundle archive:

```
<base>/target/custom-bundle/<md5(bundleName)>/codeql-bundle[-<platform>].tar.gz
```

Use `--platform linux64|osx64|win64` (repeatable) to emit platform-specific archives.

## GitHub Actions scaffolding

```
qlt action init init-test --language cpp          # per-language unit test workflow
qlt action init init-bundle-test --language cpp   # bundle integration test workflow
```

Templates live under [internal/template/files/](internal/template/files/) and are embedded at build time.

# Repository Layout Assumptions

QLT is opinionated about how the target repo is structured. At the root:

- `codeql-workspace.yml` — CodeQL workspace manifest.
- `qlt.conf.json` — QLT configuration (CodeQL version, pack bundle flags, etc.).

Queries are grouped by language, then by pack, with parallel `src/` and `test/` packs:

```
Repo Root
│   codeql-workspace.yml
│   qlt.conf.json
│
└───cpp
    ├───package1
    │   ├───src
    │   │   │   qlpack.yml
    │   │   │
    │   │   └───TestQuery
    │   │           TestQuery.ql
    │   │
    │   └───test
    │       │   qlpack.yml
    │       │
    │       └───TestQuery
    │               TestQuery.cpp
    │               TestQuery.expected
    │               TestQuery.qlref
    │
    └───package2
        └── …
```

# Configuration

`qlt.conf.json` is written to `--base` by `qlt codeql set version` (and indirectly by `qlt phase init`):

```json
{
  "codeQLCLI": "2.25.1",
  "codeQLPackConfiguration": [
    { "name": "scope/pack-name", "bundle": true }
  ]
}
```

- `codeQLCLI` is the version string for the CodeQL CLI to install and use.
- `codeQLPackConfiguration` — upserted by `qlt pack generate new-query`; entries with `bundle=true` are picked up by `qlt phase package`.

# Path Layout

Versioned directories under `$HOME/.qlt/` use an MD5 hash of the version string:

```
$HOME/.qlt/
├── packages/<md5(cliVersion)>/     ← extracted CLI
├── bundle/<md5(bundleName)>/       ← extracted bundle
```

Custom bundles produced by `qlt phase package` live under `<base>/target/custom-bundle/<md5(bundleName)>/`.

# Contributing

Contributions welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).

Commits must follow the [Conventional Commits](https://www.conventionalcommits.org/) specification.

# License

This project is released under the MIT License. See [LICENSE](LICENSE) for details.
