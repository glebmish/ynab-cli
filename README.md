# ynab-cli

A command-line interface for the [YNAB (You Need A Budget) API](https://api.ynab.com/). 100% API coverage across 44 operations. Designed for AI agents and human operators.

## Install

```bash
go install github.com/glebmish/ynab-cli@latest
```

## Configure

```bash
ynab config init
```

This writes `~/.config/ynab/config.yaml` with your personal access token (issue one at https://app.ynab.com/settings/developer) and your default `plan_id` (budget). `plan_id: last-used` works if you always want the most recently opened budget.

You can also set `YNAB_ACCESS_TOKEN`, `YNAB_PLAN_ID`, `YNAB_BASE_URL`, or `YNAB_CONFIG` in the environment.

## Examples

```bash
# Read — current month's budget summary
ynab months get --month current \
  --fields month.income,month.budgeted,month.activity,month.to_be_budgeted

# Write — create a transaction (always --dry-run first)
ynab transactions create --dry-run --json '{
  "transaction": {
    "account_id": "abc-123",
    "date": "2026-04-18",
    "amount": -42500,
    "payee_name": "Corner Store",
    "memo": "Groceries"
  }
}'
```

Amounts are **milliunits** — `$10.00 = 10000`, `-$42.50 = -42500`.

## Install AI agent skills

```bash
ynab install-skills
```

Installs structured SKILL.md files so Claude Code (or any agent) knows how to drive the CLI safely.

## Docs

- [CONTEXT.md](./CONTEXT.md) — agent-facing cheat sheet
- [CLAUDE.md](./CLAUDE.md) — repo architecture / contribution notes
- `ynab schema --list` — all 44 operations
- `ynab <resource> <action> --help` — per-command flags
