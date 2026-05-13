---
name: ynab-transactions
description: "ynab CLI: Transactions (list/get/create/update/import/delete) and scheduled transactions."
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

# Transactions & Scheduled Transactions

> **PREREQUISITE:** Read `../ynab-shared/SKILL.md` for auth, global flags, envelope unwrap, milliunit amounts, and security rules.

## transactions

All transaction operations. Filtered list views share the same flags: `--since-date YYYY-MM-DD`, `--type uncategorized|unapproved`, `--last-knowledge-of-server`.

```bash
ynab transactions <action> [flags]
```

| Action | Description |
|--------|-------------|
| `list` | List transactions (`--since-date`, `--type`, `--last-knowledge-of-server`) |
| `get` | Get a transaction (`--transaction-id`) |
| `create` | Create single or bulk transactions (`--json`) |
| `update` | Update a single transaction (`--transaction-id`, `--json`) |
| `update-bulk` | Update multiple transactions (`--json`) |
| `delete` | Delete a transaction (`--transaction-id`) |
| `import` | Trigger bank import for all linked accounts (no body) |
| `list-by-account` | List transactions for an account (`--account-id`, + list flags) |
| `list-by-category` | List transactions for a category (`--category-id`, + list flags) |
| `list-by-payee` | List transactions for a payee (`--payee-id`, + list flags) |
| `list-by-month` | List transactions for a month (`--month`, + list flags) |

### Amounts (milliunits)

All `amount` fields are in **milliunits**. `-$12.34` is `-12340`. Positive = inflow, negative = outflow.

### cleared / approved state

Each transaction carries two workflow flags:

| Field | Values | Meaning |
|---|---|---|
| `cleared` | `cleared`, `uncleared`, `reconciled` | Bank-matching state |
| `approved` | `true`, `false` | Whether the user has reviewed/approved it |

Freshly imported transactions typically arrive `cleared: "cleared"` and `approved: false` — list them with `--type unapproved` to triage.

### import_id semantics

`import_id` is a client-supplied deduplication key. Uniqueness is scoped to the account. If you try to create a transaction with an `import_id` already present on that account, YNAB returns **HTTP 409 Conflict** and the record is silently skipped.

Recommended convention: `"YNAB:<milliunits>:<YYYY-MM-DD>:<occurrence>"` where `<occurrence>` is the 1-based index of this amount/date pair in the import batch.

### Reading split transactions

A split transaction carries `category_name: "Split"` (or its split-category id) on the **parent**, with the real per-category breakdown inside `.subtransactions[]`. The parent's `amount` is the total; each subtransaction has its own signed portion.

**Any category- or payee-level analysis that operates on the parent is wrong.** A common case is paycheck splitting — a single Split transaction each payday holds subtransactions to `Tax`, `Inflow: Ready to Assign`, savings, etc. Summing by the parent's `category_name` sees zero tax and zero income.

Two ways to handle this:

**1. `--flatten-splits` (preferred for analysis).** The CLI will emit one record per subtransaction, carrying parent fields (date, account, cleared, approved, flag, etc):

```bash
ynab transactions list --since-date 2024-01-01 --flatten-splits --format ndjson > txns.ndjson
# now jq-aggregating by .category_name is correct
jq -s 'group_by(.category_name) | map({cat: .[0].category_name, total: (map(.amount_currency) | add)})' txns.ndjson
```

Semantics:
- Non-split transactions pass through unchanged.
- For split transactions, subtransaction fields override parent fields where both are set. Null subtransaction fields fall back to the parent's value (YNAB convention: null means "same as parent").
- Deleted subtransactions are dropped.
- The emitted record's `id` is the subtransaction's id; `transaction_id` points to the parent (standard YNAB shape).

**2. jq flatten (when you already have a non-flattened dump).** Equivalent logic:

```jq
[.[] | . as $p |
 if (.subtransactions | length) > 0
 then (.subtransactions[] | select(.deleted==false)
       | {date: $p.date, account_id: $p.account_id, account_name: $p.account_name,
          approved: $p.approved, cleared: $p.cleared, payee_name: ($p.payee_name),
          amount: (.amount/1000), category_id: .category_id,
          category_name: .category_name, memo: .memo,
          transfer_account_id: .transfer_account_id})
 else {date: .date, account_id: .account_id, account_name: .account_name,
       approved: .approved, cleared: .cleared, payee_name: .payee_name,
       amount: .amount_currency, category_id: .category_id,
       category_name: .category_name, memo: .memo,
       transfer_account_id: .transfer_account_id}
 end]
```

### Examples

```bash
# Triage unapproved imports for this month (always use --fields)
ynab transactions list-by-month --month current --type unapproved \
  --fields id,date,amount_currency,payee_name,category_name,memo

# Inspect schema before writing
ynab schema transactions.create --resolve-refs

# Create a single transaction (-$42.50 expense)
ynab transactions create --dry-run --json '{
  "transaction": {
    "account_id": "abc-123",
    "date": "2026-04-18",
    "amount": -42500,
    "payee_name": "Corner Store",
    "category_id": "cat-456",
    "memo": "Groceries",
    "cleared": "cleared",
    "approved": true,
    "import_id": "YNAB:-42500:2026-04-18:1"
  }
}'

# Create a split transaction (subtransactions sum to parent amount)
ynab transactions create --dry-run --json '{
  "transaction": {
    "account_id": "abc-123",
    "date": "2026-04-18",
    "amount": -60000,
    "payee_name": "Supermarket",
    "subtransactions": [
      {"amount": -40000, "category_id": "cat-groceries", "memo": "Food"},
      {"amount": -20000, "category_id": "cat-household", "memo": "Cleaning supplies"}
    ]
  }
}'

# Bulk update by import_id (pass id: null to match on import_id instead)
ynab transactions update-bulk --dry-run --json '{
  "transactions": [
    {"id": null, "import_id": "YNAB:-42500:2026-04-18:1", "approved": true, "category_id": "cat-456"}
  ]
}'

# Approve a single transaction
ynab transactions update --transaction-id t-1 --dry-run --json '{"transaction": {"approved": true}}'
```

### `import` (bank sync trigger)

```bash
ynab transactions import
```

POSTs to `/plans/{plan_id}/transactions/import` with no body. The response lists `transaction_ids` newly imported. Follow up with `ynab transactions list --type unapproved` to review them.

### Handling HTTP 409 (duplicate import_id)

If `create` or `update-bulk` returns `409 Conflict`, the CLI surfaces the error. The record was not written because its `import_id` already exists on the target account. Either (a) fetch the existing record via `list-by-account` + filter on `import_id`, or (b) regenerate the `import_id` with a higher occurrence suffix.

## scheduled-transactions

Recurring transaction templates. Read-only `list`/`get` plus standard CRUD.

```bash
ynab scheduled-transactions <action> [flags]
```

| Action | Description |
|--------|-------------|
| `list` | List scheduled transactions (`--last-knowledge-of-server`) |
| `get` | Get a scheduled transaction (`--scheduled-transaction-id`) |
| `create` | Create a scheduled transaction (`--json`) |
| `update` | Update a scheduled transaction (`--scheduled-transaction-id`, `--json`) |
| `delete` | Delete a scheduled transaction (`--scheduled-transaction-id`) |

### Examples

```bash
ynab scheduled-transactions list --fields id,date_next,amount,frequency,payee_name

ynab schema scheduled-transactions.create --resolve-refs
ynab scheduled-transactions create --dry-run --json '{
  "scheduled_transaction": {
    "account_id": "abc-123",
    "date": "2026-05-01",
    "amount": -120000,
    "payee_name": "Gym Membership",
    "category_id": "cat-subs",
    "frequency": "monthly"
  }
}'
```

## Discovering Commands

```bash
ynab schema --list
ynab schema transactions.update-bulk --resolve-refs
ynab schema TransactionDetail
```

> [!CAUTION]
> **Write commands** (`create`, `update`, `update-bulk`, `delete`, `import`) — always use `--dry-run` first and confirm with the user. Remember amounts are milliunits, and 409 means the `import_id` already exists.
