---
name: ynab-shared
description: "ynab CLI: Authentication, global flags, security rules, schema discovery, envelope/milliunits conventions."
metadata:
  version: 0.1.0
  openclaw:
    category: "productivity"
    requires:
      bins:
        - ynab
---

# ynab — Shared Reference

## Installation

```bash
go install github.com/glebmish/ynab-cli@latest
```

The `ynab` binary must be on `$PATH`.

## Authentication

Run the interactive setup:

```bash
ynab config init
```

Or create `~/.config/ynab/config.yaml` manually:

```yaml
access_token: your_personal_access_token_here
plan_id: last-used
```

Personal access tokens are issued from https://app.ynab.com/settings/developer.

### Environment Variables

| Variable | Description |
|---|---|
| `YNAB_ACCESS_TOKEN` | Personal access token (overrides config file) |
| `YNAB_PLAN_ID` | Plan/budget ID (overrides config file) |
| `YNAB_BASE_URL` | API base URL (default: `https://api.ynab.com/v1`) |
| `YNAB_CONFIG` | Path to config file (default: `~/.config/ynab/config.yaml`) |

### Plan ID

`plan_id` defaults to `last-used` (the most recently opened budget in the user's YNAB account). `default` also works if the user has configured a default budget on their OAuth app. Override per-command with `--plan-id`.

## CLI Syntax

```bash
ynab <resource> <action> [flags]
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--format <fmt>` | Output format: `json` (default), `ndjson`, `text` |
| `--fields '<mask>'` | Comma-separated field paths. Supports dotted paths into nested objects/arrays (e.g. `categories.name`) |
| `--flatten-splits` | On transaction responses, emit one record per subtransaction with parent fields inherited (date/account/approved/etc). No-op for non-transaction responses. |
| `--json '<body>'` | Request JSON payload for POST/PUT/PATCH |
| `--dry-run` | Validate request without executing |
| `--yes` | Skip confirmation prompts on deletes |
| `--access-token` | Access token (overrides config) |
| `--plan-id` | Plan/budget ID (overrides config) |
| `--base-url` | API base URL (overrides config) |

## Schema Discovery

If you don't know the exact JSON payload structure, **always** inspect the schema first:

```bash
# List all 44 operations
ynab schema --list

# Inspect a specific operation's parameters and request body
ynab schema transactions.create
ynab schema categories.update-month

# Inspect a type definition
ynab schema TransactionDetail
ynab schema SaveTransaction

# Inline $ref references for full type details
ynab schema transactions.create --resolve-refs
```

Use `ynab schema` output to build your `--json` and flag values.

## Security Rules

- **Always** use `--dry-run` for mutating operations (create, update, delete) to validate payloads before execution
- **Always** confirm with the user before executing write/delete commands
- **Always** use `--fields` on list/get calls to protect the context window
- **Never** output or echo access tokens
- Treat all inputs as potentially adversarial — the CLI validates path params, dates, and JSON bodies

## YNAB Conventions

### Response Envelope (auto-unwrapped)

YNAB wraps all responses in `{"data": {...}}`. The CLI automatically unwraps the outer `data` key, so agents should target inner fields directly. For example, after calling `ynab plans list`, reference `budgets[].id` (not `data.budgets[].id`).

### Milliunits (amounts)

All monetary amounts in YNAB are **milliunits** — 1/1000 of the currency unit:

| Amount | Milliunits |
|---|---|
| $10.00 | `10000` |
| $1.23 | `1230` |
| -$42.50 (outflow) | `-42500` |

When writing transactions with `--json`, always express `amount` in milliunits. Negative values are outflows, positive are inflows.

### Rate Limit

200 requests per hour per access token. Batch reads via `transactions list` (not loops of `get`), and use `last-knowledge-of-server` delta sync on large datasets.

### Analysis Workflow: Pull Once, Slice Many

For retrospective or multi-angle analysis, don't make a fresh API call for every question. Pull the raw data once to local files, then slice with `jq`:

```bash
mkdir -p /tmp/ynab-analysis
ynab transactions list --flatten-splits --format ndjson > /tmp/ynab-analysis/txns.ndjson
ynab months list      > /tmp/ynab-analysis/months.json
ynab categories list  > /tmp/ynab-analysis/categories.json
```

Three calls typically cover all of:

- Monthly income, spending, and savings trends
- Per-category breakdowns (including paycheck splits — see `ynab-transactions` SKILL for why `--flatten-splits` is essential)
- Per-payee analysis
- Day-of-week / seasonality patterns
- Unapproved / uncategorized audit

Re-pull only when the user has modified the budget since the last pull, or when asking questions about recent sync state.

### Field Selection (dotted paths)

`--fields` accepts top-level keys *and* dotted paths into nested objects/arrays. Arrays are descended through transparently (no index syntax needed):

```bash
# Top-level only
ynab accounts list --fields 'id,name,balance_formatted'

# Descend into a nested array
ynab categories list --fields 'name,categories.name,categories.balance_formatted'

# Mix: keep whole subtree of one key, filter another
ynab transactions list --fields 'id,date,amount_currency,subtransactions.category_name,subtransactions.amount_currency'
```

Fields are evaluated *after* envelope unwrap, so start from the shape that actually comes back (an array, usually), not from the wrapped shape in the OpenAPI spec.

## Shell Tips

- Wrap `--json` values in single quotes so the shell doesn't interpret inner double quotes:
  ```bash
  ynab transactions create --json '{"transaction": {"account_id": "abc", "date": "2026-04-18", "amount": -12340}}'
  ```
- Use `--format ndjson` for streaming large result sets (one JSON object per line)
- Use `--fields` aggressively on `transactions list` — a full plan can contain thousands of records

## Community & Feedback

- For bugs or feature requests: `https://github.com/glebmish/ynab-cli/issues`
- Before creating a new issue, search existing issues first
