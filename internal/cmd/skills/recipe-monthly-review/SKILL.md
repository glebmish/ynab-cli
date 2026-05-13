---
name: recipe-monthly-review
description: "Review the current month's budget: allocations, activity, balances, unapproved transactions."
metadata:
  version: 0.1.0
  openclaw:
    category: "recipe"
    domain: "budgeting"
    requires:
      bins:
        - ynab
      skills:
        - ynab-budgeting
        - ynab-transactions
---

# Monthly Budget Review

> **PREREQUISITE:** Load the following skills to execute this recipe: `ynab-budgeting`, `ynab-transactions`

Pull the current month's summary, category allocations, and unapproved transactions, then surface overspending vs surplus. Adapt the month to the user's request (default: `current`).

## Steps

1. **Get the month summary** (income, budgeted, activity, to-be-budgeted):
   ```bash
   ynab months get --month current \
     --fields month.month,month.income,month.budgeted,month.activity,month.to_be_budgeted,month.age_of_money
   ```

2. **Get per-category allocations and balances:**
   ```bash
   ynab categories list \
     --fields category_groups.name,category_groups.categories.name,category_groups.categories.budgeted,category_groups.categories.activity,category_groups.categories.balance
   ```

3. **Triage unapproved transactions for the month:**
   ```bash
   ynab transactions list-by-month --month current --type unapproved \
     --fields transactions.id,transactions.date,transactions.amount,transactions.payee_name,transactions.category_name,transactions.memo
   ```

4. **(Optional) Inspect any overspent categories in detail:**
   ```bash
   ynab categories get-month --category-id <CAT_ID> --month current \
     --fields category.name,category.budgeted,category.activity,category.balance
   ynab transactions list-by-category --category-id <CAT_ID> --since-date <FIRST_OF_MONTH> \
     --fields transactions.date,transactions.amount,transactions.payee_name,transactions.memo
   ```

### Summarize for the user

Remember all amounts are milliunits — divide by 1000 when narrating.

- Income vs outflow for the month; current **to_be_budgeted**
- Overspent categories (`balance < 0`): list them with the shortfall and top payees
- Underspent categories with large surplus (candidates for moving money)
- Count of unapproved transactions needing review
- Age of money trend if notable

## Tips

- `--month current` is a server-side alias for the current calendar month — always prefer it over hardcoding a date.
- Use `--format ndjson` for long category lists if you want to stream/pipe to a tool.
- If the user asks "did I overspend on X?", combine `categories get-month` + `transactions list-by-category --since-date`.
- Remember amounts are milliunits: `$10.00 = 10000`. Divide by 1000 for display.
