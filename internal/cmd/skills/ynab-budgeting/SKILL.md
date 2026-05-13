---
name: ynab-budgeting
description: "ynab CLI: User, plans, accounts, categories, payees, payee-locations, months, money-movements."
metadata:
  version: 0.1.0
  openclaw:
    category: "productivity"
    requires:
      bins:
        - ynab
      skills:
        - ynab-shared
---

# Budget Structure — Plans, Accounts, Categories, Payees, Months

> **PREREQUISITE:** Read `../ynab-shared/SKILL.md` for auth, global flags, envelope unwrap, milliunit amounts, and security rules.

## user

Authenticated user info.

```bash
ynab user get
```

| Action | Description |
|--------|-------------|
| `get` | Get the authenticated user |

## plans

Budgets (YNAB calls them "budgets"; the CLI uses `plans` throughout). `plan_id` defaults to `last-used`.

```bash
ynab plans <action> [flags]
```

| Action | Description |
|--------|-------------|
| `list` | List all plans (`--include-accounts`) |
| `get` | Get a plan by ID (uses `--plan-id` or config; `--last-knowledge-of-server`) |
| `get-settings` | Get plan settings |

### Examples

```bash
ynab plans list --fields budgets.id,budgets.name,budgets.last_modified_on
ynab plans get --fields budget.id,budget.name,budget.currency_format
```

## accounts

Budget accounts (checking, credit, savings, etc.).

```bash
ynab accounts <action> [flags]
```

| Action | Description |
|--------|-------------|
| `list` | List accounts (`--last-knowledge-of-server`) |
| `get` | Get an account (`--account-id`) |
| `create` | Create an account (`--json`) |

### Examples

```bash
ynab accounts list --fields accounts.id,accounts.name,accounts.type,accounts.balance
ynab accounts get --account-id abc-123 --fields account.id,account.name,account.balance

# Inspect schema before creating
ynab schema accounts.create --resolve-refs
ynab accounts create --dry-run --json '{"account": {"name": "Emergency Fund", "type": "savings", "balance": 500000}}'
```

Note: `balance` is in **milliunits** ($500.00 = `500000`).

## categories

Budget categories and category groups. Some operations target a specific month.

```bash
ynab categories <action> [flags]
```

| Action | Description |
|--------|-------------|
| `list` | List all categories grouped by category group (`--last-knowledge-of-server`) |
| `get` | Get a category (`--category-id`) |
| `create` | Create a category (`--json`) |
| `update` | Update a category (`--category-id`, `--json`) |
| `get-month` | Get a category for a specific month (`--category-id`, `--month`) |
| `update-month` | Update a category's budgeted amount for a month (`--category-id`, `--month`, `--json`) |
| `create-group` | Create a category group (`--json`) |
| `update-group` | Update a category group (`--category-group-id`, `--json`) |

### Examples

```bash
# Overview of current-month budget allocations
ynab categories list --fields category_groups.name,category_groups.categories.name,category_groups.categories.balance,category_groups.categories.budgeted

# Get a category scoped to the current month
ynab categories get-month --category-id c-1 --month current --fields category.name,category.budgeted,category.activity,category.balance

# Reassign $50.00 to a category for the current month
ynab categories update-month --dry-run \
  --category-id c-1 --month current \
  --json '{"category": {"budgeted": 50000}}'
```

`--month` accepts `YYYY-MM-DD` (first of month) or the literal string `current`.

## payees

Payees are the counterparties of transactions.

```bash
ynab payees <action> [flags]
```

| Action | Description |
|--------|-------------|
| `list` | List payees (`--last-knowledge-of-server`) |
| `get` | Get a payee (`--payee-id`) |
| `create` | Create a payee (`--json`) |
| `update` | Update a payee (`--payee-id`, `--json`) |

### Examples

```bash
ynab payees list --fields payees.id,payees.name,payees.transfer_account_id
ynab payees update --payee-id p-1 --dry-run --json '{"payee": {"name": "Corrected Name"}}'
```

## payee-locations

Geolocation hints for payees (used by the mobile app).

```bash
ynab payee-locations <action> [flags]
```

| Action | Description |
|--------|-------------|
| `list` | List all payee locations |
| `get` | Get a payee location (`--payee-location-id`) |
| `list-by-payee` | List locations for a specific payee (`--payee-id`) |

## months

Per-month plan summaries (income, budgeted, activity, to-be-budgeted).

```bash
ynab months <action> [flags]
```

| Action | Description |
|--------|-------------|
| `list` | List months (`--last-knowledge-of-server`) |
| `get` | Get a specific month (`--month`, `YYYY-MM-DD` or `current`) |

### Examples

```bash
ynab months get --month current --fields month.month,month.income,month.budgeted,month.activity,month.to_be_budgeted
ynab months list --fields months.month,months.to_be_budgeted
```

All amounts are milliunits.

## money-movements

Read-only view of moved money (Reserve/Move Money actions) and their groupings.

```bash
ynab money-movements <action> [flags]
```

| Action | Description |
|--------|-------------|
| `list` | List money movements across all months |
| `list-by-month` | List money movements for a month (`--month`) |
| `list-groups` | List money movement groups |
| `list-groups-by-month` | List movement groups for a month (`--month`) |

## Discovering Commands

```bash
ynab schema --list                          # All 44 operations
ynab schema categories.update-month --resolve-refs
ynab schema SaveTransaction
```

> [!CAUTION]
> **Write commands** (`create`, `update`, `update-month`, `create-group`, `update-group`) — always use `--dry-run` first and confirm with the user before executing. Remember amounts are in milliunits.
