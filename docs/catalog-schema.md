# Catalog schema

The tool catalog is a single YAML file at `internal/catalog/data/tools.yaml`,
embedded into the binary with `go:embed` and validated at load time. Any
violation fails fast with an error naming the offending tool ID.

## Top-level structure

```yaml
categories:
  - {id: utilities, name: Utilities}

profiles:
  - id: devops
    name: DevOps
    categories: [kubernetes, git, package_managers]

tools:
  - id: ripgrep
    name: ripgrep
    category: utilities
    detect: {cmd: "rg --version", regex: 'ripgrep (\d+\.\d+\.\d+)'}
    official_url: "https://github.com/BurntSushi/ripgrep"
```

## `categories`

| Field | Required | Notes |
|-------|----------|-------|
| `id` | yes | unique; referenced by tools and profiles |
| `name` | yes | display name in the TUI tree |

## `profiles`

Profiles are pure view filters over categories (the `p` key in the TUI).

| Field | Required | Notes |
|-------|----------|-------|
| `id` | yes | unique |
| `name` | yes | display name |
| `categories` | yes | list of category IDs; every entry must reference an existing category |

## `tools`

| Field | Required | Notes |
|-------|----------|-------|
| `id` | yes | unique across the catalog; the default package name managers use to synthesize update commands (`brew upgrade <id>`, `cargo install <id> --force`, ...) and to query the latest version |
| `name` | yes | display name |
| `category` | yes | must reference an existing category ID |
| `package_name` | no | overrides `id` as the name passed to the owning package manager, for tools whose crate/formula/npm package name differs from their catalog ID or binary name. Defaults to `id` when omitted |
| `detect.cmd` | yes | command run to detect the installed version (first word is also the binary looked up on `$PATH` for ownership detection) |
| `detect.regex` | yes | must compile and contain **exactly one** capture group, which extracts the version from the command's combined stdout+stderr |
| `update` | no | only for tools with a **bespoke native updater** (e.g. `rustup update`). Omit it for manager-owned tools: their update command is synthesized at runtime from the owning manager. Omit it too for tools with no updater at all |
| `update.cmd` | if `update` present | the native update command line |
| `official_url` | yes | install page opened for not-installed / no-updater rows. When it points at `https://github.com/<org>/<repo>`, it is also used to resolve the latest version via the GitHub releases redirect for manually installed tools |

## Validation rules

- no duplicate category, profile, or tool IDs
- `detect.regex` compiles and has exactly one capture group
- `tools[].category` references an existing category ID
- `profiles[].categories` entries reference existing category IDs
- `official_url` and `detect.cmd` present on every tool
- an `update` block, when present, must have a non-empty `cmd`
