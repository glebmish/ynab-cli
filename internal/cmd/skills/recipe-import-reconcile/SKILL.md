---
name: recipe-import-reconcile
description: "Trigger bank import, surface unapproved/duplicate transactions, walk the user through approval."
metadata:
  version: 0.1.0
  openclaw:
    category: "recipe"
    domain: "budgeting"
    requires:
      bins:
        - ynab
      skills:
        - ynab-transactions
---

# Import & Reconcile

> **PREREQUISITE:** Load the following skills to execute this recipe: `ynab-transactions`

Trigger YNAB's bank-import job, then help the user review new transactions and resolve duplicates (HTTP 409s from prior manual imports).

## Steps

1. **Trigger the bank import:**
   ```bash
   ynab transactions import
   ```
   The response includes `transaction_ids` for anything freshly imported. Note the count.

2. **Fetch the unapproved queue** (always use `--fields`):
   ```bash
   ynab transactions list --type unapproved \
     --fields transactions.id,transactions.date,transactions.amount,transactions.payee_name,transactions.category_name,transactions.account_name,transactions.memo,transactions.import_id
   ```

3. **Group and present to the user:**
   - By account (so they can match against a bank statement)
   - Flag any with missing `category_id`/`category_name` as needing categorization
   - Flag any with `amount` matching a prior transaction on the same date/account as a potential duplicate

4. **For each transaction the user wants to approve / edit**, use the schema first, then `--dry-run`:
   ```bash
   ynab schema transactions.update --resolve-refs
   ynab transactions update --transaction-id <ID> --dry-run \
     --json '{"transaction": {"approved": true, "category_id": "<CAT_ID>"}}'
   ```

5. **Bulk-approve the clean ones** in one call:
   ```bash
   ynab transactions update-bulk --dry-run --json '{
     "transactions": [
       {"id": "t-1", "approved": true},
       {"id": "t-2", "approved": true, "category_id": "cat-groceries"}
     ]
   }'
   ```
   Drop `--dry-run` only after the user confirms.

## Handling duplicates (HTTP 409)

If a prior manual `transactions create` returned **409 Conflict**, the `import_id` already exists on that account. Diagnose with:

```bash
ynab transactions list-by-account --account-id <ACCT> --since-date <DATE> \
  --fields transactions.id,transactions.date,transactions.amount,transactions.import_id,transactions.memo
```

Search the output for the conflicting `import_id`. Options:

- **Keep the existing record** — if it's correct, discard the attempted duplicate.
- **Update the existing record** — `ynab transactions update --transaction-id <ID> --json '{...}'`.
- **Re-create with a new `import_id`** — bump the occurrence suffix (e.g. `:2` instead of `:1`).

## Tips

- Remember amounts are milliunits (`-42500` is -$42.50). Divide by 1000 for narration.
- `--type unapproved` and `--type uncategorized` are the two main triage filters.
- The user may want to reconcile an account balance against their statement — after approving, prompt them to open the YNAB app's reconcile flow (not exposed via API).
- For large queues, `--format ndjson` pipes cleanly into downstream tooling.
