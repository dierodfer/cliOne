# CLIOne

CLIOne is a cross-platform TUI (Linux/macOS) that gives you visibility into the
developer tools installed on your machine, organized by category, and updates
them using each tool's **own native update mechanism**. It never installs
anything from scratch: for tools that are not installed, or that have no native
updater, it simply shows a link to the official install page.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Usage

```sh
make build      # builds bin/clione
./bin/clione    # opens the TUI (no subcommands in v0.1)
```

The tree shows every catalog tool grouped by category, with a 4-state
semaphore per row:

| Icon | Meaning |
|------|---------|
| 🟢 | installed and up to date |
| 🟡 | installed, a newer version is available |
| 🔴 | not installed (row shows the official install page) |
| ⚪ | installed, but no native update path is known |

Rows also show the detected version, the newest known version when an update
is available, and which package manager owns the binary (homebrew, cargo, npm,
uv, apt/dnf, or manual).

### Keys

| Key | Action |
|-----|--------|
| `↑`/`↓` (or `k`/`j`) | navigate |
| `enter` / `→` | expand/collapse a category, or act on a tool row |
| `u` | same as enter on a tool row: run the update (🟡) or open the official page (🔴/⚪) |
| `/` | fuzzy text filter (enter to keep, esc to clear) |
| `p` | cycle profile (Backend / Frontend / DevOps / AI / Full Stack / All) — filters the tree by category, combinable with `/` |
| `d` | doctor view: read-only list of tools that resolve at more than one `$PATH` location, with the active one marked |
| `l` | on a row whose update failed: toggle an inline panel with the tail of stderr |
| `esc` / `←` | collapse / back / clear filter |
| `q` | quit |

### How updates work

- Tools with a bespoke native updater declared in the catalog (e.g.
  `rustup update`, `brew update`, `copilot update`) run that command.
- Tools owned by a package manager (Homebrew, cargo, npm global, uv) run the
  manager's own upgrade command (`brew upgrade <pkg>`,
  `cargo install <pkg> --force`, ...), synthesized at runtime.
- Everything else (distro packages, manual installs) gets a link to the
  official page. CLIOne never invents an install path.

Latest versions come from each manager's own metadata (crates.io, the npm
registry, PyPI, `brew info`) or, for manually installed GitHub-hosted tools,
from the `releases/latest` redirect — no GitHub REST API, no rate limits.
Results are cached in `~/.cache/clione/cache.json` (source ownership for 7
days, latest versions for 12 hours) so the tree renders instantly; rows
refresh in place asynchronously.

## Development

```sh
make build   # go build -> bin/clione
make test    # go test ./...
make vet     # go vet ./...
make lint    # golangci-lint if installed, else go vet
make run     # go run ./cmd/clione
```

The tool catalog lives in `internal/catalog/data/tools.yaml` and is embedded
into the binary; its schema is documented in
[docs/catalog-schema.md](docs/catalog-schema.md).
