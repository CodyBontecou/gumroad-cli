# gr

CLI for the Gumroad API. Designed for humans and AI agents alike.

[![CI](https://github.com/antiwork/gr/actions/workflows/ci.yml/badge.svg)](https://github.com/antiwork/gr/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/antiwork/gr)](https://github.com/antiwork/gr)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/antiwork/gr/blob/main/LICENSE)

```
$ gr products list
ID            NAME              STATUS     PRICE
abc123        Design Templates  published  $25.00
def456        Icon Pack         published  $10.00

$ gr sales list --json --jq '.sales[0].email'
"customer@example.com"
```

## Install

**Go**:

```sh
go install github.com/antiwork/gr/cmd/gr@latest
```

**Local install with man pages and completions**:

```sh
make install

# Or install into a custom prefix
make install PREFIX="$HOME/.local"
```

Under the selected `PREFIX`, `make install` places the binary in `bin/`, man pages in `share/man/man1/`, and shell completions under `share/`.

**Binary releases**: Not published yet. For now, install with `go install` or build locally with `make build`.

Homebrew packaging is not active yet.

## Quick start

```sh
# Authenticate with your Gumroad API token
gr auth login

# Or use an ephemeral token for this shell / CI job
export GR_ACCESS_TOKEN=your-token

# View your account
gr user

# List your products
gr products list

# Fetch every page of sales
gr sales list --all

# Get a sale as JSON, filter with jq
gr sales view abc123 --json --jq '.sale.email'

# Preview a refund without executing it
gr sales refund abc123 --amount-cents 500 --dry-run
```

## Commands

```
gr auth          login, status, logout
gr user          View your account info
gr products      list, view, delete, enable, disable, skus
gr sales         list, view, refund, ship, resend-receipt
gr payouts       list, view, upcoming
gr subscribers   list, view
gr licenses      verify, enable, disable, decrement, rotate
gr offer-codes   list, view, create, update, delete
gr variant-categories list, view, create, update, delete
gr variants      list, view, create, update, delete
gr custom-fields list, create, update, delete
gr webhooks      list, create, delete
gr completion    bash, zsh, fish, powershell
```

Every command has built-in help with examples: `gr <command> --help`

Man pages are available locally after `make install`. You can also generate them directly with `make man`.

## Shell completion

Generate and install shell completions directly from the command:

```sh
# Bash
source <(gr completion bash)

# Zsh
gr completion zsh > "${fpath[1]}/_gr"

# Fish
gr completion fish | source

# PowerShell
gr completion powershell | Out-String | Invoke-Expression
```

## Output modes

| Flag | Output | Use case |
|------|--------|----------|
| *(default)* | Colored tables | Human reading |
| `--json` | JSON | Programmatic access |
| `--jq <expr>` | Filtered JSON | Extract specific fields |
| `--plain` | Tab-separated, control chars escaped | Piping to `grep`/`awk` |
| `--quiet` | Minimal | Scripts |

Paginated list commands such as `sales list`, `payouts list`, and `subscribers list` accept `--all` to fetch every page automatically. Use `--page-delay 200ms` to pace large fetches when you want to be gentler on the API.

Mutation commands keep their human success messages by default, and with `--json` they emit a stable envelope:

```json
{
  "success": true,
  "message": "Product prod_123 deleted.",
  "result": { "...": "raw API response" }
}
```

If you decline a confirmation prompt, mutating commands still emit JSON in machine-readable modes with `success: false`, `cancelled: true`, and `result: null`.

## API coverage

`gr` maps 1:1 to the [Gumroad API v2](https://app.gumroad.com/api). The CLI exposes everything the API supports today — but the API has some gaps worth knowing about:

- **No product create/update** — the API routes exist but are not implemented. Products must be created and edited through the web UI. The CLI supports delete, enable, and disable which do work.
- **No analytics or audience data** — no API endpoints exist for dashboard stats, traffic, or email lists.
- **No bulk operations** — all actions are one resource at a time.
- **Limited filtering** — `gr sales list` supports date/email/product filters, but `gr products list` returns everything with no filtering.
- **Non-standard errors** — Gumroad sometimes returns `200 OK` with `success: false` in the body instead of a 4xx/5xx.
- **Loose schemas** — some numeric fields arrive as `0` or `0.0`, and optional fields may be `null` or omitted.

`gr` normalizes these quirks where it can.

As the Gumroad API expands, the CLI will grow to match. The command structure is designed to accommodate new endpoints without breaking existing usage.

## Design principles

Built following [clig.dev](https://clig.dev/) guidelines and [`gh`](https://github.com/cli/cli) conventions:

- **Human-first, machine-readable on demand** — tables by default, `--json`/`--plain` for machines
- **Secrets never in args** — `gr auth login` prompts or reads stdin, token stored with `0600` permissions
- **Headless-friendly auth** — `GR_ACCESS_TOKEN` overrides stored config for shells, agents, and CI
- **Confirm destructive ops** — interactive confirmation for delete/refund, `--yes` to skip
- **Support safe previews** — `--dry-run` shows mutating requests without executing them
- **Rewrite errors for humans** — no raw API JSON, actionable suggestions instead
- **Respect the terminal** — colors off when not TTY or `NO_COLOR` set, pager for long output

## Architecture

```
cmd/gr/main.go         Entry point
internal/
  cmd/                 Command implementations (cobra)
    auth/              Authentication (login, status, logout)
    products/          gr products list|view|delete|enable|disable
    sales/             gr sales list|view|refund|ship|resend-receipt
    ...                One package per noun
  api/                 HTTP client for Gumroad API v2
  config/              XDG-compliant config (~/.config/gr/config.json)
  output/              Table, JSON, plain, color, spinner, pager, image rendering
  prompt/              Interactive input (token, confirmations)
  cmdutil/             Global flag state
  testutil/            Shared test helpers
```

Each command follows the same pattern: parse flags, call `api.Client`, format output via `output` package. Tests use a shared HTTP mock server from `testutil`.

## Developer Notes

- Human-facing tables should be built with a command-scoped styler (`opts.Style()` via `output.NewStyledTable`) so explicit flags like `--no-color` do not get lost behind terminal auto-detection.
- User-visible `--all` JSON/JQ output is staged before being copied to stdout. This is intentional: the CLI prefers atomic, valid output over true first-byte streaming so late failures do not leave partial JSON behind. Small responses stay in memory; larger ones spill to a temp file only when needed.

## AI agents

`gr` is designed to be used by AI coding agents. The `--json`, `--jq`, and `--no-input` flags make it easy to query Gumroad data programmatically without interactive prompts, and `GR_ACCESS_TOKEN` gives agents a no-persistence auth path.

A [Claude Code skill](.claude/skills/gr-cli/SKILL.md) is included in this repo. It teaches Claude when and how to use `gr` for Gumroad lookups — install it to let Claude automatically reach for `gr` when you ask about your products, sales, or subscribers.

## Development

```sh
make build        # Compile to ./gr
make install      # Install binary, man pages, and shell completions
make test         # Run all tests
make test-cover   # Run tests with per-package coverage gates
make test-smoke   # Run opt-in live API smoke test for read-only auth/list/view/output paths
make lint         # Run golangci-lint
make man          # Generate man pages
make snapshot     # Build release snapshot via goreleaser
```

Live smoke test:

```sh
GR_ACCESS_TOKEN=your-token make test-smoke
```

`make test-smoke` runs a small live, read-only sanity check against the real API only when `GR_SMOKE=1`; the make target sets that flag for you. It covers auth, representative list/view commands, and machine-readable output modes. Destructive flows still rely on mocked integration tests. You can optionally point it at another base URL with `GR_API_BASE_URL`.

In GitHub Actions, the same smoke suite runs automatically on non-PR workflows when the repository secret `GR_SMOKE_ACCESS_TOKEN` is configured.

Built with Go, [cobra](https://github.com/spf13/cobra), and [gojq](https://github.com/itchyny/gojq).

## License

[MIT](LICENSE)
